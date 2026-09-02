package httpserver

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"pz-web-backend/internal/config"
	"pz-web-backend/internal/gamequery"
)

// systemStats 容器内 CPU / 内存使用情况(读取 /proc)。
type systemStats struct {
	CPUPercent       float64 `json:"cpu_percent"`
	MemTotalMB       int64   `json:"mem_total_mb"`
	MemUsedMB        int64   `json:"mem_used_mb"`
	MemPercent       float64 `json:"mem_percent"`
	UptimeSec        int64   `json:"uptime_sec"`
	GameVersion      string  `json:"game_version"`
	OnlinePlayers    int     `json:"online_players"`
	MaxPlayers       int     `json:"max_players"`
	PlayerQueryOK    bool    `json:"player_query_ok"`
	PlayerQueryError string  `json:"player_query_error,omitempty"`
}

// handleSystemStats 返回容器内 CPU 与内存占用，供前端仪表盘展示。
func (a App) handleSystemStats(c *gin.Context) {
	stats, err := readSystemStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	stats.GameVersion = readGameVersion(a.BaseDataDir, a.LogPath)
	queryContext, cancel := context.WithTimeout(c.Request.Context(), 750*time.Millisecond)
	defer cancel()
	queryPort := resolveGameQueryPort(a.ConfigApp.GetServerConfig("EN"))
	if count, queryErr := gamequery.QueryPlayerCount(queryContext, net.JoinHostPort("127.0.0.1", strconv.Itoa(queryPort))); queryErr == nil {
		stats.OnlinePlayers = count.Online
		stats.MaxPlayers = count.Max
		stats.PlayerQueryOK = true
	} else {
		stats.PlayerQueryError = queryErr.Error()
	}
	c.JSON(http.StatusOK, stats)
}

func resolveGameQueryPort(items []config.Item, err error) int {
	const defaultPort = 16261
	if err != nil {
		return defaultPort
	}
	for _, item := range items {
		if item.Key != "DefaultPort" {
			continue
		}
		port, parseErr := strconv.Atoi(strings.TrimSpace(item.Value))
		if parseErr == nil && port > 0 && port <= 65535 {
			return port
		}
		break
	}
	return defaultPort
}

var gameVersionPattern = regexp.MustCompile(`version=([0-9]+(?:\.[0-9]+)+)`)

// readGameVersion 从最新服务端日志读取实际运行版本，而不是读取目标分支配置。
func readGameVersion(baseDataDir, logPath string) string {
	paths := make([]string, 0, 16)
	if logPath != "" {
		paths = append(paths, logPath)
	}
	logsDir := filepath.Join(baseDataDir, "Logs")
	_ = filepath.WalkDir(logsDir, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".txt") {
			paths = append(paths, path)
		}
		return nil
	})

	// 优先读取最近修改的日志，避免旧版本日志覆盖当前版本。
	type candidate struct {
		path string
		mod  int64
	}
	candidates := make([]candidate, 0, len(paths))
	seen := map[string]bool{}
	for _, path := range paths {
		if seen[path] {
			continue
		}
		seen[path] = true
		info, err := os.Stat(path)
		if err == nil {
			candidates = append(candidates, candidate{path: path, mod: info.ModTime().UnixNano()})
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].mod > candidates[j].mod })
	for _, item := range candidates {
		if version := versionFromLogFile(item.path); version != "" {
			return version
		}
	}
	return "unknown"
}

func versionFromLogFile(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	if info, err := f.Stat(); err == nil && info.Size() > 1024*1024 {
		_, _ = f.Seek(-1024*1024, io.SeekEnd)
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return ""
	}
	matches := gameVersionPattern.FindAllSubmatch(data, -1)
	if len(matches) == 0 {
		return ""
	}
	return string(matches[len(matches)-1][1])
}

func readSystemStats() (systemStats, error) {
	var s systemStats

	// ----- 内存 (/proc/meminfo) -----
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		fieldsMap := map[string]int64{}
		for _, line := range strings.Split(string(data), "\n") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if v, e := strconv.ParseInt(parts[1], 10, 64); e == nil {
					fieldsMap[strings.TrimSuffix(parts[0], ":")] = v
				}
			}
		}
		totalKB := fieldsMap["MemTotal"]
		availKB := fieldsMap["MemAvailable"]
		if totalKB > 0 {
			s.MemTotalMB = totalKB / 1024
			s.MemUsedMB = (totalKB - availKB) / 1024
			s.MemPercent = float64(s.MemUsedMB) / float64(s.MemTotalMB) * 100
		}
	}

	// ----- CPU (/proc/stat 两次采样差值) -----
	t1, idle1, err := readCPUTicks()
	if err == nil {
		time.Sleep(200 * time.Millisecond)
		t2, idle2, err2 := readCPUTicks()
		if err2 == nil && t2 > t1 {
			busy := (t2 - t1) - (idle2 - idle1)
			s.CPUPercent = float64(busy) / float64(t2-t1) * 100
			if s.CPUPercent > 100 {
				s.CPUPercent = 100
			}
		}
	}

	// ----- 运行时间 -----
	if data, err := os.ReadFile("/proc/uptime"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) > 0 {
			if sec, e := strconv.ParseFloat(fields[0], 64); e == nil {
				s.UptimeSec = int64(sec)
			}
		}
	}

	return s, nil
}

// readCPUTicks 返回 (总 ticks, idle ticks)。
func readCPUTicks() (uint64, uint64, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		var user, nice, system, idle uint64
		_, err := fmt.Sscanf(line, "cpu  %d %d %d %d", &user, &nice, &system, &idle)
		if err != nil {
			return 0, 0, err
		}
		total := user + nice + system + idle
		return total, idle, nil
	}
	return 0, 0, fmt.Errorf("cpu line not found")
}

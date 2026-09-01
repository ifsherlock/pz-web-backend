package httpserver

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// systemStats 容器内 CPU / 内存使用情况(读取 /proc)。
type systemStats struct {
	CPUPercent float64 `json:"cpu_percent"`
	MemTotalMB int64   `json:"mem_total_mb"`
	MemUsedMB  int64   `json:"mem_used_mb"`
	MemPercent float64 `json:"mem_percent"`
	UptimeSec  int64   `json:"uptime_sec"`
}

// handleSystemStats 返回容器内 CPU 与内存占用，供前端仪表盘展示。
func (a App) handleSystemStats(c *gin.Context) {
	stats, err := readSystemStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
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
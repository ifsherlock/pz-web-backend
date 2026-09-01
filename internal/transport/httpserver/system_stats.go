package httpserver

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// systemStats 服务器资源快照。
type systemStats struct {
	CPUPercent     float64 `json:"cpu_percent"`
	MemTotalMB     int64   `json:"mem_total_mb"`
	MemUsedMB      int64   `json:"mem_used_mb"`
	MemPercent     float64 `json:"mem_percent"`
	UptimeSeconds  int64   `json:"uptime_seconds"`
}

var (
	statsMu      sync.Mutex
	lastCPUIdle  uint64
	lastCPUTotal uint64
	lastCPUTime  time.Time
)

// collectSystemStats 读取 /proc 获取 CPU/内存占用。
// CPU 百分比基于两次采样差，保证准确性。
func collectSystemStats() systemStats {
	stats := systemStats{}

	// 内存
	if total, avail := readMemInfo(); total > 0 {
		stats.MemTotalMB = total
		stats.MemUsedMB = total - avail
		stats.MemPercent = float64(stats.MemUsedMB) / float64(total) * 100
	}

	// CPU
	if idle, total := readCPUStat(); total > 0 {
		statsMu.Lock()
		if lastCPUTotal > 0 && !lastCPUTime.IsZero() {
			dt := float64(time.Since(lastCPUTime).Milliseconds())
			dIdle := idle - lastCPUIdle
			dTotal := total - lastCPUTotal
			if dt > 0 && dTotal > 0 {
				// 归一化到相同时间片
				usage := (1 - float64(dIdle)/float64(dTotal)) * 100
				if usage < 0 {
					usage = 0
				}
				if usage > 100 {
					usage = 100
				}
				stats.CPUPercent = usage
			}
		}
		lastCPUIdle = idle
		lastCPUTotal = total
		lastCPUTime = time.Now()
		statsMu.Unlock()
	}

	// 容器/进程运行时长(取面板自身)
	stats.UptimeSeconds = readUptime()

	return stats
}

// readMemInfo 返回 (总内存MB, 可用内存MB)。
func readMemInfo() (totalMB, availMB int64) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		val, _ := strconv.ParseInt(fields[1], 10, 64)
		switch key {
		case "MemTotal":
			totalMB = val / 1024
		case "MemAvailable":
			availMB = val / 1024
		}
		if totalMB > 0 && availMB > 0 {
			break
		}
	}
	return
}

// readCPUStat 返回 (idle, total) CPU jiffies。
func readCPUStat() (idle, total uint64) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return 0, 0
	}
	fields := strings.Fields(scanner.Text())
	if len(fields) < 8 || fields[0] != "cpu" {
		return 0, 0
	}
	var values [8]uint64
	for i := 1; i <= 8 && i < len(fields); i++ {
		values[i-1], _ = strconv.ParseUint(fields[i], 10, 64)
	}
	idle = values[3] // idle
	for _, v := range values {
		total += v
	}
	return
}

// readUptime 读取系统开机时长(秒)。
func readUptime() int64 {
	f, err := os.Open("/proc/uptime")
	if err != nil {
		return 0
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return 0
	}
	fields := strings.Fields(scanner.Text())
	if len(fields) == 0 {
		return 0
	}
	sec, _ := strconv.ParseFloat(fields[0], 64)
	return int64(sec)
}

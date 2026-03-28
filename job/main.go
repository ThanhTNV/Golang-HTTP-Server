package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

/**
 * @Author: ThanhTNV
 * @Date: 2026-03-28 10:00:00
 * @LastEditors: ThanhTNV
 * @LastEditTime: 2026-03-28 10:00:00
 * @Description: Job service to run background tasks:
 * - Checking current node cpu and memory usage, and log to persistent storage
 */

const (
	logDir  = "/data"
	logFile = logDir + "/job.log"
)

func main() {
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		panic(fmt.Sprintf("failed to create log directory: %v", err))
	}

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		panic(fmt.Sprintf("failed to open log file: %v", err))
	}
	defer file.Close()

	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	multi := zerolog.MultiLevelWriter(consoleWriter, file)
	logger := zerolog.New(multi).With().Timestamp().Logger()

	logger.Info().Msg("Job service starting...")

	logMemoryUsage(&logger)
	logCPUUsage(&logger)

	logger.Info().Msg("Job service completed")
}

func logMemoryUsage(logger *zerolog.Logger) {
	memInfo, err := readProcFile("/proc/meminfo")
	if err != nil {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		logger.Warn().Err(err).Msg("Failed to read /proc/meminfo, using runtime stats")
		logger.Info().
			Uint64("alloc_mb", m.Alloc/1024/1024).
			Uint64("sys_mb", m.Sys/1024/1024).
			Uint32("num_gc", m.NumGC).
			Msg("Go runtime memory usage")
		return
	}

	totalKB := parseMemInfoValue(memInfo, "MemTotal")
	freeKB := parseMemInfoValue(memInfo, "MemFree")
	availKB := parseMemInfoValue(memInfo, "MemAvailable")
	buffersKB := parseMemInfoValue(memInfo, "Buffers")
	cachedKB := parseMemInfoValue(memInfo, "Cached")
	usedKB := totalKB - freeKB - buffersKB - cachedKB

	logger.Info().
		Uint64("total_mb", totalKB/1024).
		Uint64("used_mb", usedKB/1024).
		Uint64("free_mb", freeKB/1024).
		Uint64("available_mb", availKB/1024).
		Uint64("buffers_mb", buffersKB/1024).
		Uint64("cached_mb", cachedKB/1024).
		Str("usage_percent", fmt.Sprintf("%.1f%%", float64(usedKB)/float64(totalKB)*100)).
		Msg("Node memory usage")
}

func logCPUUsage(logger *zerolog.Logger) {
	idle1, total1, err := readCPUStat()
	if err != nil {
		logger.Warn().Err(err).
			Int("num_cpu", runtime.NumCPU()).
			Msg("Failed to read /proc/stat, reporting CPU count only")
		return
	}

	time.Sleep(1 * time.Second)

	idle2, total2, err := readCPUStat()
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to read /proc/stat on second sample")
		return
	}

	idleDelta := float64(idle2 - idle1)
	totalDelta := float64(total2 - total1)
	cpuUsage := (1.0 - idleDelta/totalDelta) * 100

	logger.Info().
		Int("num_cpu", runtime.NumCPU()).
		Str("usage_percent", fmt.Sprintf("%.1f%%", cpuUsage)).
		Msg("Node CPU usage")
}

func readProcFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), ":", 2)
		if len(parts) == 2 {
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return result, scanner.Err()
}

func parseMemInfoValue(info map[string]string, key string) uint64 {
	val, ok := info[key]
	if !ok {
		return 0
	}
	fields := strings.Fields(val)
	if len(fields) == 0 {
		return 0
	}
	v, _ := strconv.ParseUint(fields[0], 10, 64)
	return v
}

func readCPUStat() (idle, total uint64, err error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			return 0, 0, fmt.Errorf("unexpected /proc/stat format")
		}
		for i := 1; i < len(fields); i++ {
			v, _ := strconv.ParseUint(fields[i], 10, 64)
			total += v
			if i == 4 {
				idle = v
			}
		}
		return idle, total, nil
	}
	return 0, 0, fmt.Errorf("cpu line not found in /proc/stat")
}

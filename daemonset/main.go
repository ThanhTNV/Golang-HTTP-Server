package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
)

/**
 * @Author: ThanhTNV
 * @Date: 2026-03-28 10:00:00
 * @LastEditors: ThanhTNV
 * @LastEditTime: 2026-03-28 10:00:00
 * @Description: DaemonSet service to run background tasks:
 * - Checking current node cpu, memory usage, and node network usage
 * - Responding to HTTP requests with the current node status
 */

type NodeStatus struct {
	Hostname string        `json:"hostname"`
	CPU      CPUStatus     `json:"cpu"`
	Memory   MemoryStatus  `json:"memory"`
	Network  []NetDevStats `json:"network"`
}

type CPUStatus struct {
	NumCPU       int     `json:"num_cpu"`
	UsagePercent float64 `json:"usage_percent"`
}

type MemoryStatus struct {
	MemoryUsage string `json:"memoryUsage"`
	MemoryTotal string `json:"memoryTotal"`
	MemoryFree  string `json:"memoryFree"`
	RSS         string `json:"rss"`
}

type NetDevStats struct {
	Interface string `json:"interface"`
	RxBytes   uint64 `json:"rx_bytes"`
	RxPackets uint64 `json:"rx_packets"`
	RxErrors  uint64 `json:"rx_errors"`
	TxBytes   uint64 `json:"tx_bytes"`
	TxPackets uint64 `json:"tx_packets"`
	TxErrors  uint64 `json:"tx_errors"`
}

func main() {
	app := fiber.New()

	app.Get("/health", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	app.Get("/status", func(c fiber.Ctx) error {
		status, err := collectNodeStatus()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.JSON(status)
	})

	app.Get("/status/cpu", func(c fiber.Ctx) error {
		cpu, err := collectCPU()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.JSON(cpu)
	})

	app.Get("/status/memory", func(c fiber.Ctx) error {
		mem, err := collectMemory()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.JSON(mem)
	})

	app.Get("/status/network", func(c fiber.Ctx) error {
		net, err := collectNetwork()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.JSON(net)
	})

	app.Listen(":3000")
}

func collectNodeStatus() (*NodeStatus, error) {
	hostname, _ := os.Hostname()

	cpu, err := collectCPU()
	if err != nil {
		return nil, fmt.Errorf("cpu: %w", err)
	}

	mem, err := collectMemory()
	if err != nil {
		return nil, fmt.Errorf("memory: %w", err)
	}

	net, err := collectNetwork()
	if err != nil {
		return nil, fmt.Errorf("network: %w", err)
	}

	return &NodeStatus{
		Hostname: hostname,
		CPU:      *cpu,
		Memory:   *mem,
		Network:  net,
	}, nil
}

func collectCPU() (*CPUStatus, error) {
	idle1, total1, err := readCPUStat()
	if err != nil {
		return nil, err
	}

	time.Sleep(500 * time.Millisecond)

	idle2, total2, err := readCPUStat()
	if err != nil {
		return nil, err
	}

	totalDelta := float64(total2 - total1)
	idleDelta := float64(idle2 - idle1)
	usage := 0.0
	if totalDelta > 0 {
		usage = (1.0 - idleDelta/totalDelta) * 100
	}

	return &CPUStatus{
		NumCPU:       runtime.NumCPU(),
		UsagePercent: math_round(usage, 2),
	}, nil
}

func collectMemory() (*MemoryStatus, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	toMB := func(b uint64) string {
		return fmt.Sprintf("%.2f MB", float64(b)/1024/1024)
	}

	return &MemoryStatus{
		MemoryUsage: toMB(m.HeapInuse),
		MemoryTotal: toMB(m.HeapSys),
		MemoryFree:  toMB(m.HeapIdle),
		RSS:         toMB(m.Sys),
	}, nil
}

func collectNetwork() ([]NetDevStats, error) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var stats []NetDevStats
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum <= 2 {
			continue // skip header lines
		}

		line := scanner.Text()
		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}

		iface := strings.TrimSpace(line[:colonIdx])
		fields := strings.Fields(line[colonIdx+1:])
		if len(fields) < 16 {
			continue
		}

		stats = append(stats, NetDevStats{
			Interface: iface,
			RxBytes:   parseUint(fields[0]),
			RxPackets: parseUint(fields[1]),
			RxErrors:  parseUint(fields[2]),
			TxBytes:   parseUint(fields[8]),
			TxPackets: parseUint(fields[9]),
			TxErrors:  parseUint(fields[10]),
		})
	}
	return stats, scanner.Err()
}

// --- /proc helpers ---

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

func parseUint(s string) uint64 {
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}

func math_round(val float64, precision int) float64 {
	p := 1.0
	for i := 0; i < precision; i++ {
		p *= 10
	}
	return float64(int(val*p+0.5)) / p
}

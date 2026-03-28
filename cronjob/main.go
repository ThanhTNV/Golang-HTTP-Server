package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

/**
 * @Author: ThanhTNV
 * @Date: 2026-03-28 10:00:00
 * @LastEditors: ThanhTNV
 * @LastEditTime: 2026-03-28 10:00:00
 * @Description: Cronjob service to run background tasks:
 * - Calling to daemonset service to get node status
 * - Logging the node status to persistent storage
 */

const (
	logDir  = "/data"
	logFile = logDir + "/cronjob.log"

	saTokenPath  = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	saCACertPath = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	saNamespace  = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

	daemonsetServiceName = "k8s-pod-daemonset-svc"
	daemonsetPort        = 3000
)

// Kubernetes Endpoints API response
type Endpoints struct {
	Subsets []Subset `json:"subsets"`
}

type Subset struct {
	Addresses []Address `json:"addresses"`
}

type Address struct {
	IP       string  `json:"ip"`
	NodeName *string `json:"nodeName"`
}

// DaemonSet /status response (mirrors daemonset/main.go types)
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

	logger.Info().Msg("CronJob service starting...")

	endpoints, err := discoverEndpoints()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to discover daemonset endpoints")
	}

	logger.Info().Int("count", len(endpoints)).Msg("Discovered daemonset pods")

	client := &http.Client{Timeout: 10 * time.Second}
	for _, ep := range endpoints {
		nodeName := ""
		if ep.NodeName != nil {
			nodeName = *ep.NodeName
		}

		url := fmt.Sprintf("http://%s:%d/status", ep.IP, daemonsetPort)
		logger.Info().Str("node", nodeName).Str("ip", ep.IP).Msg("Querying node status")

		status, err := queryStatus(client, url)
		if err != nil {
			logger.Error().Err(err).Str("node", nodeName).Str("ip", ep.IP).Msg("Failed to query node status")
			continue
		}

		logNodeStatus(&logger, nodeName, status)
	}

	logger.Info().Msg("CronJob service completed")
}

// discoverEndpoints queries the Kubernetes Endpoints API using the
// in-cluster service account to find all DaemonSet pod IPs.
func discoverEndpoints() ([]Address, error) {
	token, err := os.ReadFile(saTokenPath)
	if err != nil {
		return nil, fmt.Errorf("read SA token: %w", err)
	}

	caCert, err := os.ReadFile(saCACertPath)
	if err != nil {
		return nil, fmt.Errorf("read CA cert: %w", err)
	}

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caCert)

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: pool},
		},
	}

	namespace := readFileOr(saNamespace, "default")
	url := fmt.Sprintf("https://kubernetes.default.svc/api/v1/namespaces/%s/endpoints/%s", namespace, daemonsetServiceName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+string(token))

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("K8s API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("K8s API returned %d: %s", resp.StatusCode, body)
	}

	var ep Endpoints
	if err := json.NewDecoder(resp.Body).Decode(&ep); err != nil {
		return nil, fmt.Errorf("decode endpoints: %w", err)
	}

	var addrs []Address
	for _, subset := range ep.Subsets {
		addrs = append(addrs, subset.Addresses...)
	}
	return addrs, nil
}

func queryStatus(client *http.Client, url string) (*NodeStatus, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, body)
	}

	var status NodeStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}
	return &status, nil
}

func logNodeStatus(logger *zerolog.Logger, nodeName string, s *NodeStatus) {
	logger.Info().
		Str("node", nodeName).
		Str("hostname", s.Hostname).
		Int("cpu_count", s.CPU.NumCPU).
		Float64("cpu_usage_pct", s.CPU.UsagePercent).
		Str("mem_usage", s.Memory.MemoryUsage).
		Str("mem_total", s.Memory.MemoryTotal).
		Str("mem_free", s.Memory.MemoryFree).
		Str("rss", s.Memory.RSS).
		Int("network_interfaces", len(s.Network)).
		Msg("Node status collected")

	for _, n := range s.Network {
		logger.Info().
			Str("node", nodeName).
			Str("interface", n.Interface).
			Uint64("rx_bytes", n.RxBytes).
			Uint64("tx_bytes", n.TxBytes).
			Uint64("rx_errors", n.RxErrors).
			Uint64("tx_errors", n.TxErrors).
			Msg("Network interface stats")
	}
}

func readFileOr(path, fallback string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return fallback
	}
	return strings.TrimSpace(string(data))
}

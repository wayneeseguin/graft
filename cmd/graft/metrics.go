package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/wayneeseguin/graft/internal"
	"github.com/wayneeseguin/graft/log"
)

// StartMetricsServer starts the metrics server if enabled
func StartMetricsServer() *internal.MetricsServer {
	features := internal.GetFeatures()

	if !features.EnableMetrics {
		return nil
	}

	port := features.MetricsPort
	if portStr := os.Getenv("GRAFT_METRICS_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	metrics := internal.GetParallelMetrics()
	server := internal.NewMetricsServer(metrics, port)

	if err := server.Start(); err != nil {
		log.DEBUG("Failed to start metrics server: %v", err)
		return nil
	}

	fmt.Fprintf(os.Stderr, "Metrics server started on port %d\n", port)
	fmt.Fprintf(os.Stderr, "  Prometheus metrics: http://localhost:%d/metrics\n", port)
	fmt.Fprintf(os.Stderr, "  JSON metrics: http://localhost:%d/metrics/json\n", port)
	fmt.Fprintf(os.Stderr, "  Health check: http://localhost:%d/health\n", port)

	return server
}

// PrintMetricsSummary prints a summary of execution metrics
func PrintMetricsSummary() {
	features := internal.GetFeatures()

	if !features.EnableMetrics {
		return
	}

	metrics := internal.GetParallelMetrics()
	snapshot := metrics.GetSnapshot()

	if snapshot.OpsTotal == 0 {
		return
	}

	fmt.Fprintf(os.Stderr, "\n=== Execution Metrics ===\n")
	fmt.Fprintf(os.Stderr, "%s\n", snapshot.String())
	fmt.Fprintf(os.Stderr, "========================\n")
}

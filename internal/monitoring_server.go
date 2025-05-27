package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sync"
	"time"
)

// MonitoringServer provides HTTP endpoints for monitoring
type MonitoringServer struct {
	mu             sync.RWMutex
	server         *http.Server
	metricsReg     *MetricsRegistry
	cacheAnalytics *CacheAnalytics
	timingAgg      *TimingAggregator
	slowOpDetector *SlowOperationDetector
	profiler       *Profiler
	config         *MonitoringConfig
}

// MonitoringConfig configures the monitoring server
type MonitoringConfig struct {
	Enabled         bool
	ListenAddr      string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	EnableDashboard bool
}

// DefaultMonitoringConfig returns default monitoring configuration
func DefaultMonitoringConfig() *MonitoringConfig {
	return &MonitoringConfig{
		Enabled:         true,
		ListenAddr:      ":8080",
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    10 * time.Second,
		EnableDashboard: true,
	}
}

// NewMonitoringServer creates a new monitoring server
func NewMonitoringServer(config *MonitoringConfig) *MonitoringServer {
	if config == nil {
		config = DefaultMonitoringConfig()
	}

	return &MonitoringServer{
		config:         config,
		metricsReg:     GetMetricsRegistry(),
		cacheAnalytics: GetCacheAnalytics(),
		timingAgg:      GetTimingAggregator(),
		slowOpDetector: GetSlowOpDetector(),
		profiler:       GetProfiler(),
	}
}

// Start starts the monitoring server
func (ms *MonitoringServer) Start() error {
	if !ms.config.Enabled {
		return nil
	}

	mux := http.NewServeMux()

	// Metrics endpoints
	mux.HandleFunc("/metrics", ms.handleMetrics)
	mux.HandleFunc("/metrics/json", ms.handleMetricsJSON)

	// Analytics endpoints
	mux.HandleFunc("/analytics/cache", ms.handleCacheAnalytics)
	mux.HandleFunc("/analytics/timing", ms.handleTimingAnalytics)
	mux.HandleFunc("/analytics/slow-ops", ms.handleSlowOps)

	// Health endpoint
	mux.HandleFunc("/health", ms.handleHealth)

	// Dashboard
	if ms.config.EnableDashboard {
		mux.HandleFunc("/", ms.handleDashboard)
		mux.HandleFunc("/api/metrics/stream", ms.handleMetricsStream)
	}

	ms.server = &http.Server{
		Addr:         ms.config.ListenAddr,
		Handler:      mux,
		ReadTimeout:  ms.config.ReadTimeout,
		WriteTimeout: ms.config.WriteTimeout,
	}

	go func() {
		if err := ms.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Monitoring server error: %v\n", err)
		}
	}()

	return nil
}

// Stop stops the monitoring server
func (ms *MonitoringServer) Stop() error {
	if ms.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return ms.server.Shutdown(ctx)
}

// handleMetrics handles Prometheus metrics endpoint
func (ms *MonitoringServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	exporter := NewMetricsExporter(ms.metricsReg)
	if err := exporter.ExportPrometheus(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleMetricsJSON handles JSON metrics endpoint
func (ms *MonitoringServer) handleMetricsJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	exporter := NewMetricsExporter(ms.metricsReg)
	if err := exporter.ExportJSON(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleCacheAnalytics handles cache analytics endpoint
func (ms *MonitoringServer) handleCacheAnalytics(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")

	reporter := NewCacheReporter(ms.cacheAnalytics)

	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		report := ms.cacheAnalytics.GenerateReport()
		json.NewEncoder(w).Encode(report)

	case "metrics":
		w.Header().Set("Content-Type", "text/plain")
		reporter.GenerateMetricsReport(w)

	default:
		w.Header().Set("Content-Type", "text/plain")
		reporter.GenerateTextReport(w)
	}
}

// handleTimingAnalytics handles timing analytics endpoint
func (ms *MonitoringServer) handleTimingAnalytics(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")

	if format == "json" {
		w.Header().Set("Content-Type", "application/json")
		summary := ms.timingAgg.GetSummary()
		json.NewEncoder(w).Encode(summary)
	} else {
		w.Header().Set("Content-Type", "text/plain")
		stats := ms.timingAgg.GetAllStats()

		fmt.Fprintf(w, "=== Operation Timing Statistics ===\n\n")
		for _, stat := range stats {
			fmt.Fprintf(w, "%s\n", stat.String())
		}
	}
}

// handleSlowOps handles slow operations endpoint
func (ms *MonitoringServer) handleSlowOps(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	history := ms.slowOpDetector.GetHistory(limit)
	stats := ms.slowOpDetector.history.GetStats()

	format := r.URL.Query().Get("format")
	if format == "json" {
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"stats":   stats,
			"history": history,
		}
		json.NewEncoder(w).Encode(response)
	} else {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "=== Slow Operations ===\n")
		fmt.Fprintf(w, "Total: %d\n\n", stats.Total)

		for op, typeStats := range stats.ByOperation {
			fmt.Fprintf(w, "%s: %d occurrences, avg: %v, max: %v\n",
				op, typeStats.Count, typeStats.AvgTime, typeStats.MaxTime)
		}

		if len(history) > 0 {
			fmt.Fprintf(w, "\nRecent Slow Operations:\n")
			for i, op := range history {
				if i >= 10 {
					break
				}
				fmt.Fprintf(w, "- %s: %v (threshold: %v) at %s\n",
					op.Name, op.Duration, op.Threshold, op.Start.Format("15:04:05"))
			}
		}
	}
}

// handleHealth handles health check endpoint
func (ms *MonitoringServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"uptime":    time.Since(ms.metricsReg.startTime),
		"version":   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// handleDashboard handles the dashboard page
func (ms *MonitoringServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>Graft Performance Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        h1 { color: #333; }
        .container { max-width: 1200px; margin: 0 auto; }
        .metric-card { 
            background: white; 
            border-radius: 8px; 
            padding: 20px; 
            margin: 10px 0; 
            box-shadow: 0 2px 4px rgba(0,0,0,0.1); 
        }
        .metric-title { font-size: 18px; font-weight: bold; color: #555; }
        .metric-value { font-size: 36px; font-weight: bold; color: #2196F3; }
        .metric-unit { font-size: 14px; color: #888; }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; }
        .chart { height: 300px; background: #fafafa; border-radius: 4px; margin-top: 20px; }
        .nav { 
            background: white; 
            padding: 15px; 
            border-radius: 8px; 
            margin-bottom: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .nav a { 
            margin-right: 20px; 
            text-decoration: none; 
            color: #2196F3; 
            font-weight: bold;
        }
        .nav a:hover { text-decoration: underline; }
        #metrics { white-space: pre-wrap; font-family: monospace; }
        .refresh-info { text-align: right; color: #888; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Graft Performance Dashboard</h1>
        
        <div class="nav">
            <a href="/">Dashboard</a>
            <a href="/metrics">Prometheus Metrics</a>
            <a href="/metrics/json">JSON Metrics</a>
            <a href="/analytics/cache">Cache Analytics</a>
            <a href="/analytics/timing">Timing Analytics</a>
            <a href="/analytics/slow-ops">Slow Operations</a>
            <a href="/health">Health</a>
        </div>
        
        <div class="grid" id="overview">
            <div class="metric-card">
                <div class="metric-title">Operations/sec</div>
                <div class="metric-value" id="ops-per-sec">--</div>
                <div class="metric-unit">ops/s</div>
            </div>
            
            <div class="metric-card">
                <div class="metric-title">Cache Hit Rate</div>
                <div class="metric-value" id="cache-hit-rate">--</div>
                <div class="metric-unit">%</div>
            </div>
            
            <div class="metric-card">
                <div class="metric-title">Memory Usage</div>
                <div class="metric-value" id="memory-usage">--</div>
                <div class="metric-unit">MB</div>
            </div>
            
            <div class="metric-card">
                <div class="metric-title">Goroutines</div>
                <div class="metric-value" id="goroutines">--</div>
                <div class="metric-unit">active</div>
            </div>
        </div>
        
        <div class="metric-card">
            <div class="metric-title">Real-time Metrics</div>
            <div class="refresh-info">Auto-refreshing every 5 seconds</div>
            <div id="metrics"></div>
        </div>
    </div>
    
    <script>
        function updateMetrics() {
            fetch('/metrics/json')
                .then(resp => resp.json())
                .then(data => {
                    // Update overview cards
                    const opsPerSec = getMetricValue(data, 'graft_operations_per_second');
                    document.getElementById('ops-per-sec').textContent = 
                        opsPerSec ? opsPerSec.toFixed(1) : '0';
                    
                    // Calculate cache hit rate
                    const cacheHits = getMetricSum(data, 'graft_cache_hits_total');
                    const cacheMisses = getMetricSum(data, 'graft_cache_misses_total');
                    const hitRate = cacheHits + cacheMisses > 0 ? 
                        (cacheHits / (cacheHits + cacheMisses) * 100) : 0;
                    document.getElementById('cache-hit-rate').textContent = hitRate.toFixed(1);
                    
                    // Memory usage
                    const heapAlloc = data.resources?.memory?.HeapAlloc || 0;
                    document.getElementById('memory-usage').textContent = 
                        (heapAlloc / 1024 / 1024).toFixed(1);
                    
                    // Goroutines
                    const goroutines = data.resources?.goroutines || 0;
                    document.getElementById('goroutines').textContent = goroutines;
                    
                    // Update detailed metrics
                    document.getElementById('metrics').textContent = 
                        JSON.stringify(data, null, 2);
                })
                .catch(err => console.error('Failed to fetch metrics:', err));
        }
        
        function getMetricValue(data, metricName) {
            const metric = data.metrics[metricName];
            if (!metric || !metric.metrics) return 0;
            const values = Object.values(metric.metrics);
            return values.length > 0 ? values[0] : 0;
        }
        
        function getMetricSum(data, metricName) {
            const metric = data.metrics[metricName];
            if (!metric || !metric.metrics) return 0;
            return Object.values(metric.metrics).reduce((sum, val) => sum + val, 0);
        }
        
        // Update immediately and then every 5 seconds
        updateMetrics();
        setInterval(updateMetrics, 5000);
    </script>
</body>
</html>`

	t, _ := template.New("dashboard").Parse(tmpl)
	t.Execute(w, nil)
}

// handleMetricsStream handles WebSocket metrics streaming
func (ms *MonitoringServer) handleMetricsStream(w http.ResponseWriter, r *http.Request) {
	// For simplicity, using SSE instead of WebSocket
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			snapshot := ms.metricsReg.GetSnapshot()
			data, _ := json.Marshal(snapshot)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

		case <-r.Context().Done():
			return
		}
	}
}

// Global monitoring server
var globalMonitoringServer *MonitoringServer
var monitoringOnce sync.Once

// InitializeMonitoringServer initializes the global monitoring server
func InitializeMonitoringServer(config *MonitoringConfig) error {
	var err error
	monitoringOnce.Do(func() {
		globalMonitoringServer = NewMonitoringServer(config)
		err = globalMonitoringServer.Start()
	})
	return err
}

// GetMonitoringServer returns the global monitoring server
func GetMonitoringServer() *MonitoringServer {
	return globalMonitoringServer
}

// StopMonitoringServer stops the global monitoring server
func StopMonitoringServer() error {
	if globalMonitoringServer != nil {
		return globalMonitoringServer.Stop()
	}
	return nil
}

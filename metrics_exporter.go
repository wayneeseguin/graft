package spruce

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// MetricsExporter exports metrics in various formats
type MetricsExporter struct {
	registry *MetricsRegistry
}

// NewMetricsExporter creates a new metrics exporter
func NewMetricsExporter(registry *MetricsRegistry) *MetricsExporter {
	return &MetricsExporter{
		registry: registry,
	}
}

// ExportPrometheus exports metrics in Prometheus format
func (e *MetricsExporter) ExportPrometheus(w io.Writer) error {
	snapshot := e.registry.GetSnapshot()
	
	// Write header
	fmt.Fprintf(w, "# Spruce metrics export\n")
	fmt.Fprintf(w, "# Timestamp: %s\n", snapshot.Timestamp.Format(time.RFC3339))
	fmt.Fprintf(w, "# Uptime: %s\n\n", snapshot.Uptime)
	
	// Export each metric family
	families := e.registry.collector.GetAllMetricFamilies()
	for _, family := range families {
		if err := e.exportPrometheusFamily(w, family); err != nil {
			return err
		}
	}
	
	// Export resource metrics
	e.exportPrometheusResources(w, snapshot.Resources)
	
	return nil
}

// exportPrometheusFamily exports a metric family in Prometheus format
func (e *MetricsExporter) exportPrometheusFamily(w io.Writer, family *MetricFamily) error {
	// Write help and type
	fmt.Fprintf(w, "# HELP %s %s\n", family.Name, family.Help)
	fmt.Fprintf(w, "# TYPE %s %s\n", family.Name, strings.ToLower(string(family.Type)))
	
	// Export each metric
	metrics := family.GetAll()
	
	// Sort metrics for consistent output
	sort.Slice(metrics, func(i, j int) bool {
		return labelsToKey(metrics[i].Labels()) < labelsToKey(metrics[j].Labels())
	})
	
	for _, metric := range metrics {
		labels := metric.Labels()
		labelStr := ""
		if len(labels) > 0 {
			pairs := make([]string, 0, len(labels))
			for k, v := range labels {
				pairs = append(pairs, fmt.Sprintf(`%s="%s"`, k, v))
			}
			sort.Strings(pairs)
			labelStr = "{" + strings.Join(pairs, ",") + "}"
		}
		
		switch m := metric.(type) {
		case *Counter:
			fmt.Fprintf(w, "%s%s %d\n", family.Name, labelStr, m.Get())
			
		case *Gauge:
			fmt.Fprintf(w, "%s%s %d\n", family.Name, labelStr, m.Get())
			
		case *Histogram:
			stats := m.GetStats()
			base := family.Name + labelStr
			
			// Export histogram stats
			fmt.Fprintf(w, "%s_sum %f\n", base, stats.Sum)
			fmt.Fprintf(w, "%s_count %d\n", base, stats.Count)
			
			// Export buckets (simplified - using percentiles as buckets)
			buckets := []struct {
				le    string
				value float64
			}{
				{"0.5", stats.P50},
				{"0.9", stats.P90},
				{"0.95", stats.P95},
				{"0.99", stats.P99},
				{"+Inf", stats.Max},
			}
			
			for _, bucket := range buckets {
				fmt.Fprintf(w, "%s_bucket{le=\"%s\"} %f\n", base, bucket.le, bucket.value)
			}
			
		case *Summary:
			// Similar to histogram but with quantiles
			stats := m.GetStats()
			base := family.Name + labelStr
			
			fmt.Fprintf(w, "%s_sum %f\n", base, stats.Sum)
			fmt.Fprintf(w, "%s_count %d\n", base, stats.Count)
			
			quantiles := m.GetQuantiles()
			for q, v := range quantiles {
				fmt.Fprintf(w, "%s{quantile=\"%g\"} %f\n", base, q, v)
			}
		}
	}
	
	fmt.Fprintln(w) // Empty line between families
	return nil
}

// exportPrometheusResources exports resource metrics in Prometheus format
func (e *MetricsExporter) exportPrometheusResources(w io.Writer, resources *ResourceSnapshot) {
	// Process info
	fmt.Fprintf(w, "# HELP spruce_process_info Spruce process information\n")
	fmt.Fprintf(w, "# TYPE spruce_process_info gauge\n")
	fmt.Fprintf(w, "spruce_process_info{cpus=\"%d\"} 1\n\n", resources.NumCPU)
	
	// Memory metrics
	fmt.Fprintf(w, "# HELP spruce_memory_bytes Memory usage by type\n")
	fmt.Fprintf(w, "# TYPE spruce_memory_bytes gauge\n")
	fmt.Fprintf(w, "spruce_memory_bytes{type=\"heap_alloc\"} %d\n", resources.MemStats.HeapAlloc)
	fmt.Fprintf(w, "spruce_memory_bytes{type=\"heap_sys\"} %d\n", resources.MemStats.HeapSys)
	fmt.Fprintf(w, "spruce_memory_bytes{type=\"heap_idle\"} %d\n", resources.MemStats.HeapIdle)
	fmt.Fprintf(w, "spruce_memory_bytes{type=\"heap_inuse\"} %d\n", resources.MemStats.HeapInuse)
	fmt.Fprintf(w, "spruce_memory_bytes{type=\"stack_sys\"} %d\n", resources.MemStats.StackSys)
	fmt.Fprintf(w, "spruce_memory_bytes{type=\"sys\"} %d\n\n", resources.MemStats.Sys)
	
	// GC metrics
	fmt.Fprintf(w, "# HELP spruce_gc_total Total number of GC cycles\n")
	fmt.Fprintf(w, "# TYPE spruce_gc_total counter\n")
	fmt.Fprintf(w, "spruce_gc_total %d\n\n", resources.MemStats.NumGC)
	
	fmt.Fprintf(w, "# HELP spruce_gc_pause_seconds GC pause durations\n")
	fmt.Fprintf(w, "# TYPE spruce_gc_pause_seconds summary\n")
	if len(resources.RecentGCPauses) > 0 {
		var sum float64
		for _, pause := range resources.RecentGCPauses {
			sum += pause.Seconds()
		}
		fmt.Fprintf(w, "spruce_gc_pause_seconds_sum %f\n", sum)
		fmt.Fprintf(w, "spruce_gc_pause_seconds_count %d\n", len(resources.RecentGCPauses))
	}
	fmt.Fprintln(w)
}

// ExportJSON exports metrics in JSON format
func (e *MetricsExporter) ExportJSON(w io.Writer) error {
	snapshot := e.registry.GetSnapshot()
	
	// Create structured output
	output := map[string]interface{}{
		"timestamp": snapshot.Timestamp,
		"uptime":    snapshot.Uptime.String(),
		"metrics":   make(map[string]interface{}),
		"resources": snapshot.Resources,
	}
	
	// Process each metric family
	for _, family := range e.registry.collector.GetAllMetricFamilies() {
		familyData := map[string]interface{}{
			"help":    family.Help,
			"type":    family.Type,
			"metrics": make(map[string]interface{}),
		}
		
		for _, metric := range family.GetAll() {
			key := labelsToKey(metric.Labels())
			if key == "" {
				key = "default"
			}
			
			value := metric.Value()
			
			// Enhance histogram/summary output
			switch family.Type {
			case MetricTypeHistogram:
				if stats, ok := value.(HistogramStats); ok {
					value = map[string]interface{}{
						"count": stats.Count,
						"sum":   stats.Sum,
						"min":   stats.Min,
						"max":   stats.Max,
						"mean":  stats.Mean,
						"percentiles": map[string]float64{
							"p50": stats.P50,
							"p90": stats.P90,
							"p95": stats.P95,
							"p99": stats.P99,
						},
					}
				}
			}
			
			familyData["metrics"].(map[string]interface{})[key] = value
		}
		
		output["metrics"].(map[string]interface{})[family.Name] = familyData
	}
	
	// Encode to JSON
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// ExportText exports metrics in human-readable text format
func (e *MetricsExporter) ExportText(w io.Writer) error {
	snapshot := e.registry.GetSnapshot()
	
	// Header
	fmt.Fprintf(w, "=== Spruce Performance Metrics ===\n")
	fmt.Fprintf(w, "Timestamp: %s\n", snapshot.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "Uptime: %s\n\n", formatMetricsDuration(snapshot.Uptime))
	
	// Parsing metrics
	fmt.Fprintf(w, "Parsing Performance:\n")
	e.exportTextSection(w, []string{
		"spruce_parse_operations_total",
		"spruce_parse_duration_seconds",
		"spruce_parse_errors_total",
	})
	
	// Evaluation metrics
	fmt.Fprintf(w, "\nEvaluation Performance:\n")
	e.exportTextSection(w, []string{
		"spruce_eval_operations_total",
		"spruce_eval_duration_seconds",
	})
	
	// Cache metrics
	fmt.Fprintf(w, "\nCache Performance:\n")
	e.exportTextCacheMetrics(w)
	
	// Resource metrics
	fmt.Fprintf(w, "\nResource Usage:\n")
	e.exportTextResources(w, snapshot.Resources)
	
	// Throughput
	fmt.Fprintf(w, "\nThroughput:\n")
	e.exportTextSection(w, []string{
		"spruce_operations_per_second",
		"spruce_documents_processed_total",
		"spruce_bytes_processed_total",
	})
	
	return nil
}

// exportTextSection exports a section of metrics in text format
func (e *MetricsExporter) exportTextSection(w io.Writer, metricNames []string) {
	for _, name := range metricNames {
		for _, family := range e.registry.collector.GetAllMetricFamilies() {
			if family.Name != name {
				continue
			}
			
			for _, metric := range family.GetAll() {
				label := ""
				if len(metric.Labels()) > 0 {
					label = " " + labelsToKey(metric.Labels())
				}
				
				switch m := metric.(type) {
				case *Counter:
					fmt.Fprintf(w, "  %s%s: %d\n", family.Name, label, m.Get())
				case *Gauge:
					fmt.Fprintf(w, "  %s%s: %d\n", family.Name, label, m.Get())
				case *Histogram:
					stats := m.GetStats()
					fmt.Fprintf(w, "  %s%s:\n", family.Name, label)
					fmt.Fprintf(w, "    Count: %d\n", stats.Count)
					fmt.Fprintf(w, "    Mean: %.3fs\n", stats.Mean)
					fmt.Fprintf(w, "    P50: %.3fs\n", stats.P50)
					fmt.Fprintf(w, "    P95: %.3fs\n", stats.P95)
					fmt.Fprintf(w, "    P99: %.3fs\n", stats.P99)
				}
			}
		}
	}
}

// exportTextCacheMetrics exports cache metrics in text format
func (e *MetricsExporter) exportTextCacheMetrics(w io.Writer) {
	cacheTypes := make(map[string]struct {
		hits   int64
		misses int64
		size   int64
	})
	
	// Collect cache metrics by type
	for _, metric := range e.registry.collector.CacheHits.GetAll() {
		if cache, ok := metric.Labels()["cache"]; ok {
			stats := cacheTypes[cache]
			stats.hits = metric.(*Counter).Get()
			cacheTypes[cache] = stats
		}
	}
	
	for _, metric := range e.registry.collector.CacheMisses.GetAll() {
		if cache, ok := metric.Labels()["cache"]; ok {
			stats := cacheTypes[cache]
			stats.misses = metric.(*Counter).Get()
			cacheTypes[cache] = stats
		}
	}
	
	for _, metric := range e.registry.collector.CacheSize.GetAll() {
		if cache, ok := metric.Labels()["cache"]; ok {
			stats := cacheTypes[cache]
			stats.size = metric.(*Gauge).Get()
			cacheTypes[cache] = stats
		}
	}
	
	// Display cache stats
	for cacheType, stats := range cacheTypes {
		total := stats.hits + stats.misses
		hitRate := float64(0)
		if total > 0 {
			hitRate = float64(stats.hits) / float64(total) * 100
		}
		
		fmt.Fprintf(w, "  %s cache:\n", cacheType)
		fmt.Fprintf(w, "    Hit rate: %.1f%% (%d hits, %d misses)\n", hitRate, stats.hits, stats.misses)
		fmt.Fprintf(w, "    Size: %d entries\n", stats.size)
	}
}

// exportTextResources exports resource metrics in text format
func (e *MetricsExporter) exportTextResources(w io.Writer, resources *ResourceSnapshot) {
	fmt.Fprintf(w, "  Memory:\n")
	fmt.Fprintf(w, "    Heap allocated: %s\n", formatBytes(resources.MemStats.HeapAlloc))
	fmt.Fprintf(w, "    Heap objects: %d\n", resources.MemStats.HeapObjects)
	fmt.Fprintf(w, "    System memory: %s\n", formatBytes(resources.MemStats.Sys))
	fmt.Fprintf(w, "  Runtime:\n")
	fmt.Fprintf(w, "    Goroutines: %d\n", resources.NumGoroutines)
	fmt.Fprintf(w, "    CPUs: %d\n", resources.NumCPU)
	fmt.Fprintf(w, "    GC runs: %d\n", resources.MemStats.NumGC)
	
	if len(resources.RecentGCPauses) > 0 {
		var total time.Duration
		var max time.Duration
		for _, pause := range resources.RecentGCPauses {
			total += pause
			if pause > max {
				max = pause
			}
		}
		avg := total / time.Duration(len(resources.RecentGCPauses))
		fmt.Fprintf(w, "    GC pause (avg): %v\n", avg)
		fmt.Fprintf(w, "    GC pause (max): %v\n", max)
	}
}

// Helper functions

func formatMetricsDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	
	parts := []string{}
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if seconds > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}
	
	return strings.Join(parts, " ")
}

func formatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// Export formats
type ExportFormat string

const (
	ExportFormatPrometheus ExportFormat = "prometheus"
	ExportFormatJSON       ExportFormat = "json"
	ExportFormatText       ExportFormat = "text"
)

// Export exports metrics in the specified format
func (e *MetricsExporter) Export(format ExportFormat) ([]byte, error) {
	var buf bytes.Buffer
	
	switch format {
	case ExportFormatPrometheus:
		err := e.ExportPrometheus(&buf)
		return buf.Bytes(), err
	case ExportFormatJSON:
		err := e.ExportJSON(&buf)
		return buf.Bytes(), err
	case ExportFormatText:
		err := e.ExportText(&buf)
		return buf.Bytes(), err
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}
package spruce

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"text/tabwriter"
	"time"
)

// CacheReporter generates cache performance reports
type CacheReporter struct {
	analytics *CacheAnalytics
}

// NewCacheReporter creates a new cache reporter
func NewCacheReporter(analytics *CacheAnalytics) *CacheReporter {
	return &CacheReporter{
		analytics: analytics,
	}
}

// GenerateTextReport generates a human-readable text report
func (cr *CacheReporter) GenerateTextReport(w io.Writer) error {
	report := cr.analytics.GenerateReport()
	
	fmt.Fprintf(w, "=== Cache Performance Report ===\n")
	fmt.Fprintf(w, "Generated: %s\n", report.GeneratedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "Period: %s\n", formatDuration(report.AnalyticsPeriod))
	fmt.Fprintf(w, "Overall Hit Rate: %.1f%%\n", report.OverallHitRate*100)
	fmt.Fprintf(w, "Effectiveness Score: %.1f/100\n\n", report.EffectivenessScore*100)
	
	// Cache statistics table
	fmt.Fprintf(w, "Cache Statistics:\n")
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "Cache\tHits\tMisses\tHit Rate\tSize\tFill Rate\tEvictions\tAvg Load")
	fmt.Fprintln(tw, "-----\t----\t------\t--------\t----\t---------\t---------\t--------")
	
	for _, stats := range report.CacheStats {
		fmt.Fprintf(tw, "%s\t%d\t%d\t%.1f%%\t%d/%d\t%.1f%%\t%d\t%v\n",
			stats.Name,
			stats.Hits,
			stats.Misses,
			stats.HitRate*100,
			stats.Size,
			stats.MaxSize,
			stats.FillRate*100,
			stats.Evictions,
			stats.AvgLoadTime.Round(time.Microsecond),
		)
	}
	tw.Flush()
	
	// Hot keys section
	if len(report.HotKeys) > 0 {
		fmt.Fprintf(w, "\nHot Keys (Top %d):\n", len(report.HotKeys))
		tw = tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "Key\tAccess Count\tHit Rate\tAvg Load Time")
		fmt.Fprintln(tw, "---\t------------\t--------\t-------------")
		
		for i, key := range report.HotKeys {
			if i >= 10 {
				break // Show only top 10
			}
			keyDisplay := key.Key
			if len(keyDisplay) > 40 {
				keyDisplay = keyDisplay[:37] + "..."
			}
			fmt.Fprintf(tw, "%s\t%d\t%.1f%%\t%v\n",
				keyDisplay,
				key.AccessCount,
				key.HitRate*100,
				key.AvgLoadTime.Round(time.Microsecond),
			)
		}
		tw.Flush()
	}
	
	// Recommendations
	fmt.Fprintf(w, "\nRecommendations:\n")
	recommendations := cr.generateRecommendations(report)
	for _, rec := range recommendations {
		fmt.Fprintf(w, "- %s\n", rec)
	}
	
	return nil
}

// GenerateCompactReport generates a compact report
func (cr *CacheReporter) GenerateCompactReport() string {
	report := cr.analytics.GenerateReport()
	
	var buf bytes.Buffer
	for _, stats := range report.CacheStats {
		fmt.Fprintf(&buf, "%s: %.1f%% hit rate (%d/%d), ",
			stats.Name,
			stats.HitRate*100,
			stats.Hits,
			stats.Hits+stats.Misses,
		)
	}
	
	result := buf.String()
	if len(result) > 2 {
		result = result[:len(result)-2] // Remove trailing ", "
	}
	
	return result
}

// GenerateMetricsReport generates a metrics-style report
func (cr *CacheReporter) GenerateMetricsReport(w io.Writer) error {
	report := cr.analytics.GenerateReport()
	
	// Header
	fmt.Fprintf(w, "# Cache Analytics Report\n")
	fmt.Fprintf(w, "# Generated at %s\n\n", report.GeneratedAt.Format(time.RFC3339))
	
	// Overall metrics
	fmt.Fprintf(w, "cache_total_hits %d\n", report.TotalHits)
	fmt.Fprintf(w, "cache_total_misses %d\n", report.TotalMisses)
	fmt.Fprintf(w, "cache_total_evictions %d\n", report.TotalEvictions)
	fmt.Fprintf(w, "cache_overall_hit_rate %.4f\n", report.OverallHitRate)
	fmt.Fprintf(w, "cache_effectiveness_score %.4f\n", report.EffectivenessScore)
	fmt.Fprintf(w, "cache_total_size %d\n", report.TotalSize)
	fmt.Fprintf(w, "cache_total_capacity %d\n", report.TotalMaxSize)
	fmt.Fprintln(w)
	
	// Per-cache metrics
	for _, stats := range report.CacheStats {
		labels := fmt.Sprintf(`{cache="%s"}`, stats.Name)
		fmt.Fprintf(w, "cache_hits%s %d\n", labels, stats.Hits)
		fmt.Fprintf(w, "cache_misses%s %d\n", labels, stats.Misses)
		fmt.Fprintf(w, "cache_evictions%s %d\n", labels, stats.Evictions)
		fmt.Fprintf(w, "cache_size%s %d\n", labels, stats.Size)
		fmt.Fprintf(w, "cache_max_size%s %d\n", labels, stats.MaxSize)
		fmt.Fprintf(w, "cache_hit_rate%s %.4f\n", labels, stats.HitRate)
		fmt.Fprintf(w, "cache_fill_rate%s %.4f\n", labels, stats.FillRate)
		fmt.Fprintf(w, "cache_avg_load_time_seconds%s %.6f\n", labels, stats.AvgLoadTime.Seconds())
		fmt.Fprintln(w)
	}
	
	// Hot key metrics
	for i, key := range report.HotKeys {
		if i >= 10 {
			break
		}
		labels := fmt.Sprintf(`{rank="%d"}`, i+1)
		fmt.Fprintf(w, "cache_hot_key_accesses%s %d\n", labels, key.AccessCount)
		fmt.Fprintf(w, "cache_hot_key_hit_rate%s %.4f\n", labels, key.HitRate)
	}
	
	return nil
}

// generateRecommendations generates cache tuning recommendations
func (cr *CacheReporter) generateRecommendations(report *CacheAnalyticsReport) []string {
	var recommendations []string
	
	// Check overall hit rate
	if report.OverallHitRate < 0.7 {
		recommendations = append(recommendations,
			fmt.Sprintf("Overall hit rate is low (%.1f%%). Consider increasing cache sizes or reviewing cache keys.",
				report.OverallHitRate*100))
	}
	
	// Check individual caches
	for _, stats := range report.CacheStats {
		// Low hit rate
		if stats.HitRate < 0.5 && stats.Hits+stats.Misses > 100 {
			recommendations = append(recommendations,
				fmt.Sprintf("%s cache has low hit rate (%.1f%%). Review cache strategy.",
					stats.Name, stats.HitRate*100))
		}
		
		// High eviction rate
		evictionRate := float64(stats.Evictions) / float64(stats.Hits+stats.Misses)
		if evictionRate > 0.1 && stats.Evictions > 100 {
			recommendations = append(recommendations,
				fmt.Sprintf("%s cache has high eviction rate (%.1f%%). Consider increasing cache size.",
					stats.Name, evictionRate*100))
		}
		
		// Underutilized cache
		if stats.FillRate < 0.3 && stats.MaxSize > 1000 {
			recommendations = append(recommendations,
				fmt.Sprintf("%s cache is underutilized (%.1f%% full). Consider reducing cache size.",
					stats.Name, stats.FillRate*100))
		}
		
		// Slow load times
		if stats.AvgLoadTime > 100*time.Millisecond {
			recommendations = append(recommendations,
				fmt.Sprintf("%s cache has slow average load time (%v). Consider optimizing data source.",
					stats.Name, stats.AvgLoadTime))
		}
	}
	
	// Hot key analysis
	if len(report.HotKeys) > 0 {
		topKey := report.HotKeys[0]
		totalAccesses := int64(0)
		for _, stats := range report.CacheStats {
			totalAccesses += stats.Hits + stats.Misses
		}
		
		if totalAccesses > 0 {
			topKeyPercentage := float64(topKey.AccessCount) / float64(totalAccesses) * 100
			if topKeyPercentage > 10 {
				recommendations = append(recommendations,
					fmt.Sprintf("Key '%s' accounts for %.1f%% of cache accesses. Consider dedicated optimization.",
						truncateKey(topKey.Key, 30), topKeyPercentage))
			}
		}
	}
	
	// Effectiveness score
	if report.EffectivenessScore < 0.6 {
		recommendations = append(recommendations,
			"Overall cache effectiveness is low. Review cache configuration and access patterns.")
	}
	
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Cache performance is good. No immediate optimizations needed.")
	}
	
	return recommendations
}

// GenerateDiffReport generates a report comparing two time periods
func (cr *CacheReporter) GenerateDiffReport(previous, current *CacheAnalyticsReport, w io.Writer) error {
	fmt.Fprintf(w, "=== Cache Performance Comparison ===\n")
	fmt.Fprintf(w, "Previous: %s\n", previous.GeneratedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "Current:  %s\n", current.GeneratedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintln(w)
	
	// Overall metrics comparison
	fmt.Fprintf(w, "Overall Metrics:\n")
	fmt.Fprintf(w, "  Hit Rate: %.1f%% → %.1f%% (%+.1f%%)\n",
		previous.OverallHitRate*100,
		current.OverallHitRate*100,
		(current.OverallHitRate-previous.OverallHitRate)*100,
	)
	fmt.Fprintf(w, "  Effectiveness: %.1f → %.1f (%+.1f)\n",
		previous.EffectivenessScore*100,
		current.EffectivenessScore*100,
		(current.EffectivenessScore-previous.EffectivenessScore)*100,
	)
	fmt.Fprintln(w)
	
	// Per-cache comparison
	fmt.Fprintf(w, "Cache Changes:\n")
	
	// Build maps for easy lookup
	prevMap := make(map[string]CacheStatistics)
	for _, stats := range previous.CacheStats {
		prevMap[stats.Name] = stats
	}
	
	currMap := make(map[string]CacheStatistics)
	for _, stats := range current.CacheStats {
		currMap[stats.Name] = stats
	}
	
	// Compare caches
	for name, currStats := range currMap {
		if prevStats, exists := prevMap[name]; exists {
			hitRateDiff := (currStats.HitRate - prevStats.HitRate) * 100
			if math.Abs(hitRateDiff) > 1.0 { // Only show significant changes
				fmt.Fprintf(w, "  %s:\n", name)
				fmt.Fprintf(w, "    Hit Rate: %.1f%% → %.1f%% (%+.1f%%)\n",
					prevStats.HitRate*100,
					currStats.HitRate*100,
					hitRateDiff,
				)
				
				if currStats.AvgLoadTime != prevStats.AvgLoadTime {
					fmt.Fprintf(w, "    Avg Load: %v → %v\n",
						prevStats.AvgLoadTime.Round(time.Microsecond),
						currStats.AvgLoadTime.Round(time.Microsecond),
					)
				}
			}
		} else {
			fmt.Fprintf(w, "  %s: NEW CACHE\n", name)
		}
	}
	
	return nil
}

// Helper functions

func truncateKey(key string, maxLen int) string {
	if len(key) <= maxLen {
		return key
	}
	return key[:maxLen-3] + "..."
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
	return fmt.Sprintf("%.1fd", d.Hours()/24)
}
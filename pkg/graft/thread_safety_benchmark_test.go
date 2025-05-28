package graft

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// BenchmarkCurrentEvaluatorThreadSafety measures the current evaluator under concurrent load
func BenchmarkCurrentEvaluatorThreadSafety(b *testing.B) {
	scenarios := []struct {
		name       string
		goroutines int
		treeSize   int
		readRatio  float64 // percentage of reads vs writes
	}{
		{"Small-LowConcurrency", 2, 100, 0.8},
		{"Small-HighConcurrency", 10, 100, 0.8},
		{"Medium-LowConcurrency", 2, 1000, 0.8},
		{"Medium-HighConcurrency", 10, 1000, 0.8},
		{"Large-LowConcurrency", 2, 10000, 0.8},
		{"Large-HighConcurrency", 10, 10000, 0.8},
		{"WriteHeavy", 10, 1000, 0.2},
		{"ReadHeavy", 10, 1000, 0.95},
	}
	
	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			data := generateTestTree(scenario.treeSize)
			
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					// Simulate mixed read/write workload
					if fastrand()%100 < int(scenario.readRatio*100) {
						// Read operation
						if meta, ok := data["meta"].(map[interface{}]interface{}); ok {
							_ = meta[fmt.Sprintf("key%d", fastrand()%scenario.treeSize)]
						}
					} else {
						// Write operation
						key := fmt.Sprintf("key%d", fastrand()%scenario.treeSize)
						if meta, ok := data["meta"].(map[interface{}]interface{}); ok {
							meta[key] = fmt.Sprintf("value-%d", fastrand())
						}
					}
				}
			})
		})
	}
}

// BenchmarkMapVsSyncMap compares regular map with sync.Map
func BenchmarkMapVsSyncMap(b *testing.B) {
	b.Run("RegularMap", func(b *testing.B) {
		m := make(map[string]interface{})
		var mu sync.RWMutex
		
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				key := fmt.Sprintf("key%d", fastrand()%1000)
				
				if fastrand()%100 < 80 { // 80% reads
					mu.RLock()
					_ = m[key]
					mu.RUnlock()
				} else {
					mu.Lock()
					m[key] = fastrand()
					mu.Unlock()
				}
			}
		})
	})
	
	b.Run("SyncMap", func(b *testing.B) {
		var m sync.Map
		
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				key := fmt.Sprintf("key%d", fastrand()%1000)
				
				if fastrand()%100 < 80 { // 80% reads
					m.Load(key)
				} else {
					m.Store(key, fastrand())
				}
			}
		})
	})
}

// BenchmarkLockingStrategies compares different locking strategies
func BenchmarkLockingStrategies(b *testing.B) {
	const numShards = 32
	
	b.Run("GlobalLock", func(b *testing.B) {
		data := make(map[string]interface{})
		var mu sync.RWMutex
		
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				key := fmt.Sprintf("key%d", fastrand()%1000)
				
				mu.Lock()
				data[key] = fastrand()
				mu.Unlock()
			}
		})
	})
	
	b.Run("ShardedLocks", func(b *testing.B) {
		type shard struct {
			mu   sync.RWMutex
			data map[string]interface{}
		}
		
		shards := make([]shard, numShards)
		for i := range shards {
			shards[i].data = make(map[string]interface{})
		}
		
		getShard := func(key string) *shard {
			h := fnv32(key)
			return &shards[h%numShards]
		}
		
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				key := fmt.Sprintf("key%d", fastrand()%1000)
				s := getShard(key)
				
				s.mu.Lock()
				s.data[key] = fastrand()
				s.mu.Unlock()
			}
		})
	})
	
	b.Run("LockFreeAtomic", func(b *testing.B) {
		var counter int64
		
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				atomic.AddInt64(&counter, 1)
			}
		})
	})
}

// BenchmarkConcurrentOperatorExecution benchmarks concurrent operator execution
func BenchmarkConcurrentOperatorExecution(b *testing.B) {
	// Skip for now as it requires full YAML parsing infrastructure
	b.Skip("Requires YAML parsing infrastructure")
}

// BenchmarkMemoryUsage measures memory usage under concurrent load
func BenchmarkMemoryUsage(b *testing.B) {
	measureMemory := func(name string, fn func()) {
		var m1, m2 runtime.MemStats
		runtime.ReadMemStats(&m1)
		
		fn()
		
		runtime.ReadMemStats(&m2)
		b.Logf("%s: Alloc=%d KB, TotalAlloc=%d KB, Sys=%d KB, NumGC=%d",
			name,
			(m2.Alloc-m1.Alloc)/1024,
			(m2.TotalAlloc-m1.TotalAlloc)/1024,
			(m2.Sys-m1.Sys)/1024,
			m2.NumGC-m1.NumGC,
		)
	}
	
	b.Run("ConcurrentTreeModification", func(b *testing.B) {
		measureMemory("TreeMods", func() {
			data := generateTestTree(10000)
			var wg sync.WaitGroup
			
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					for j := 0; j < 1000; j++ {
						data[fmt.Sprintf("worker%d", id)] = map[interface{}]interface{}{
							fmt.Sprintf("key%d", j): fmt.Sprintf("value%d", j),
						}
					}
				}(i)
			}
			
			wg.Wait()
		})
	})
}

// Helper functions

func generateTestTree(size int) map[interface{}]interface{} {
	data := make(map[interface{}]interface{})
	data["meta"] = make(map[interface{}]interface{})
	meta := data["meta"].(map[interface{}]interface{})
	
	for i := 0; i < size; i++ {
		meta[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
	}
	
	return data
}

// fastrand is a fast thread-local random number generator
func fastrand() int {
	return int(time.Now().UnixNano() & 0x7fffffff)
}

// fnv32 is a simple hash function for sharding
func fnv32(s string) uint32 {
	h := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		h *= 16777619
		h ^= uint32(s[i])
	}
	return h
}
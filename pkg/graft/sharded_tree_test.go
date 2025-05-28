package graft

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
	
	. "github.com/smartystreets/goconvey/convey"
)

func TestShardedLockManager(t *testing.T) {
	Convey("ShardedLockManager operations", t, func() {
		slm := NewShardedLockManager(4)
		
		Convey("Basic locking", func() {
			// Test that different paths can be locked concurrently
			unlock1 := slm.Lock([]string{"path1"}, false)
			unlock2 := slm.Lock([]string{"path2"}, false)
			
			unlock1()
			unlock2()
		})
		
		Convey("Shard distribution", func() {
			// Test that different paths map to different shards (statistically)
			shardCounts := make(map[uint32]int)
			
			for i := 0; i < 100; i++ {
				path := []string{fmt.Sprintf("key%d", i)}
				shardIndex := slm.getShardIndex(path)
				shardCounts[shardIndex]++
			}
			
			// Should have some distribution across shards
			So(len(shardCounts), ShouldBeGreaterThan, 1)
		})
		
		Convey("TryLock with timeout", func() {
			// Acquire a write lock
			unlock := slm.Lock([]string{"contested"}, true)
			
			// Try to acquire another write lock with timeout
			start := time.Now()
			_, err := slm.TryLock([]string{"contested"}, true, 50*time.Millisecond)
			elapsed := time.Since(start)
			
			So(err, ShouldNotBeNil)
			So(elapsed, ShouldBeGreaterThanOrEqualTo, 50*time.Millisecond)
			So(elapsed, ShouldBeLessThan, 100*time.Millisecond)
			
			unlock()
		})
	})
}

func TestShardedSafeTree(t *testing.T) {
	Convey("ShardedSafeTree operations", t, func() {
		tree := NewShardedSafeTree(nil, 8)
		
		Convey("Basic operations", func() {
			err := tree.Set("test-value", "meta", "name")
			So(err, ShouldBeNil)
			
			value, err := tree.Find("meta", "name")
			So(err, ShouldBeNil)
			So(value, ShouldEqual, "test-value")
			
			So(tree.Exists("meta", "name"), ShouldBeTrue)
			So(tree.Exists("nonexistent"), ShouldBeFalse)
		})
		
		Convey("Copy operation", func() {
			tree.Set("original", "key")
			copied := tree.Copy()
			
			// Modify original
			tree.Set("modified", "key")
			
			// Copy should remain unchanged
			value, err := copied.Find("key")
			So(err, ShouldBeNil)
			So(value, ShouldEqual, "original")
		})
		
		Convey("CompareAndSwap operation", func() {
			tree.Set("old-value", "key")
			
			// Successful swap
			success := tree.CompareAndSwap("old-value", "new-value", "key")
			So(success, ShouldBeTrue)
			
			value, _ := tree.Find("key")
			So(value, ShouldEqual, "new-value")
			
			// Failed swap (wrong old value)
			success = tree.CompareAndSwap("wrong-value", "another-value", "key")
			So(success, ShouldBeFalse)
		})
	})
}

func TestShardedTreeConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency tests in short mode")
	}
	
	Convey("ShardedSafeTree concurrent operations", t, func() {
		tree := NewShardedSafeTree(nil, 16)
		
		Convey("Concurrent operations on different shards", func() {
			var wg sync.WaitGroup
			errors := make(chan error, 100)
			
			// Multiple workers operating on different key spaces
			for worker := 0; worker < 8; worker++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					
					// Each worker uses its own key prefix to minimize contention
					prefix := fmt.Sprintf("worker%d", id)
					
					for i := 0; i < 50; i++ {
						key := fmt.Sprintf("key%d", i)
						value := fmt.Sprintf("value-%d-%d", id, i)
						
						// Write
						err := tree.Set(value, prefix, key)
						if err != nil {
							errors <- err
							return
						}
						
						// Read
						readValue, err := tree.Find(prefix, key)
						if err != nil {
							errors <- err
							return
						}
						
						if readValue != value {
							errors <- fmt.Errorf("unexpected value: got %v, want %v", readValue, value)
							return
						}
					}
				}(worker)
			}
			
			wg.Wait()
			close(errors)
			
			// Should have no errors
			var errorCount int
			for err := range errors {
				t.Logf("Concurrency error: %v", err)
				errorCount++
			}
			
			So(errorCount, ShouldEqual, 0)
		})
		
		Convey("High contention scenario", func() {
			// Multiple goroutines competing for the same counter
			tree.Set(0, "counter")
			
			var wg sync.WaitGroup
			successCount := make(chan int, 200)
			
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < 20; j++ {
						for attempts := 0; attempts < 1000; attempts++ {
							current, _ := tree.Find("counter")
							if currentVal, ok := current.(int); ok {
								if tree.CompareAndSwap(currentVal, currentVal+1, "counter") {
									successCount <- 1
									break
								}
							}
							// Use runtime.Gosched() for faster context switching
							runtime.Gosched()
						}
					}
				}()
			}
			
			wg.Wait()
			close(successCount)
			
			// Count successful operations
			total := 0
			for range successCount {
				total++
			}
			
			So(total, ShouldEqual, 200)
			
			// Final counter value should be 200
			finalValue, _ := tree.Find("counter")
			So(finalValue, ShouldEqual, 200)
		})
	})
}

func BenchmarkShardedVsBasicTree(b *testing.B) {
	// Compare performance of sharded vs basic SafeTree
	
	b.Run("Basic-Find", func(b *testing.B) {
		tree := NewSafeTree(nil)
		for i := 0; i < 1000; i++ {
			tree.Set(fmt.Sprintf("value-%d", i), "data", fmt.Sprintf("key%d", i))
		}
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("key%d", i%1000)
				tree.Find("data", key)
				i++
			}
		})
	})
	
	b.Run("Sharded-Find", func(b *testing.B) {
		tree := NewShardedSafeTree(nil, 32)
		for i := 0; i < 1000; i++ {
			tree.Set(fmt.Sprintf("value-%d", i), "data", fmt.Sprintf("key%d", i))
		}
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("key%d", i%1000)
				tree.Find("data", key)
				i++
			}
		})
	})
	
	b.Run("Basic-Set", func(b *testing.B) {
		tree := NewSafeTree(nil)
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("key%d", i)
				tree.Set(fmt.Sprintf("value-%d", i), "data", key)
				i++
			}
		})
	})
	
	b.Run("Sharded-Set", func(b *testing.B) {
		tree := NewShardedSafeTree(nil, 32)
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("key%d", i)
				tree.Set(fmt.Sprintf("value-%d", i), "data", key)
				i++
			}
		})
	})
	
	b.Run("Basic-CompareAndSwap", func(b *testing.B) {
		tree := NewSafeTree(nil)
		tree.Set(0, "counter")
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				for {
					current, _ := tree.Find("counter")
					if currentVal, ok := current.(int); ok {
						if tree.CompareAndSwap(currentVal, currentVal+1, "counter") {
							break
						}
					}
				}
			}
		})
	})
	
	b.Run("Sharded-CompareAndSwap", func(b *testing.B) {
		tree := NewShardedSafeTree(nil, 32)
		tree.Set(0, "counter")
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				for {
					current, _ := tree.Find("counter")
					if currentVal, ok := current.(int); ok {
						if tree.CompareAndSwap(currentVal, currentVal+1, "counter") {
							break
						}
					}
				}
			}
		})
	})
}
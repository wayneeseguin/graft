package internal

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSafeTree(t *testing.T) {
	Convey("SafeTree basic operations", t, func() {
		tree := NewSafeTree(nil)
		
		Convey("Set and Find operations", func() {
			err := tree.Set("test-value", "meta", "name")
			So(err, ShouldBeNil)
			
			value, err := tree.Find("meta", "name")
			So(err, ShouldBeNil)
			So(value, ShouldEqual, "test-value")
		})
		
		Convey("Exists operation", func() {
			tree.Set("value", "key")
			So(tree.Exists("key"), ShouldBeTrue)
			So(tree.Exists("nonexistent"), ShouldBeFalse)
		})
		
		Convey("Delete operation", func() {
			tree.Set("value", "key")
			So(tree.Exists("key"), ShouldBeTrue)
			
			err := tree.Delete("key")
			So(err, ShouldBeNil)
			So(tree.Exists("key"), ShouldBeFalse)
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
			
			value, _ = tree.Find("key")
			So(value, ShouldEqual, "new-value") // Should remain unchanged
		})
		
		Convey("Update operation", func() {
			tree.Set(10, "counter")
			
			err := tree.Update(func(current interface{}) interface{} {
				if val, ok := current.(int); ok {
					return val + 1
				}
				return 1
			}, "counter")
			
			So(err, ShouldBeNil)
			
			value, err := tree.Find("counter")
			So(err, ShouldBeNil)
			So(value, ShouldEqual, 11)
		})
	})
}

func TestSafeTreeConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency tests in short mode")
	}
	
	Convey("SafeTree concurrent operations", t, func() {
		tree := NewSafeTree(nil)
		
		Convey("Concurrent reads and writes", func() {
			var wg sync.WaitGroup
			errors := make(chan error, 100)
			
			// Initialize some data
			for i := 0; i < 10; i++ {
				tree.Set(fmt.Sprintf("value-%d", i), "section", fmt.Sprintf("key%d", i))
			}
			
			// Readers
			for i := 0; i < 5; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					for j := 0; j < 100; j++ {
						key := fmt.Sprintf("key%d", j%10)
						value, err := tree.Find("section", key)
						if err != nil {
							errors <- fmt.Errorf("reader %d: %v", id, err)
						} else if value == nil {
							errors <- fmt.Errorf("reader %d: nil value for %s", id, key)
						}
					}
				}(i)
			}
			
			// Writers
			for i := 0; i < 3; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					for j := 0; j < 50; j++ {
						key := fmt.Sprintf("key%d", j%10)
						value := fmt.Sprintf("writer-%d-%d", id, j)
						err := tree.Set(value, "section", key)
						if err != nil {
							errors <- fmt.Errorf("writer %d: %v", id, err)
						}
					}
				}(i)
			}
			
			wg.Wait()
			close(errors)
			
			// Should have no errors with thread-safe implementation
			var errorCount int
			for err := range errors {
				t.Logf("Concurrency error: %v", err)
				errorCount++
			}
			
			So(errorCount, ShouldEqual, 0)
		})
		
		Convey("Concurrent CompareAndSwap operations", func() {
			tree.Set(0, "counter")
			
			var wg sync.WaitGroup
			successCount := int32(0)
			
			// Multiple goroutines trying to increment the counter
			// Reduced to 5 goroutines x 20 increments = 100 total
			for i := 0; i < 5; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < 20; j++ {
						retries := 0
						for retries < 1000 { // Add max retries to prevent infinite loop
							current, _ := tree.Find("counter")
							if currentVal, ok := current.(int); ok {
								if tree.CompareAndSwap(currentVal, currentVal+1, "counter") {
									atomic.AddInt32(&successCount, 1)
									break
								}
							}
							retries++
							// Use runtime.Gosched() instead of sleep for faster context switching
							runtime.Gosched()
						}
					}
				}()
			}
			
			wg.Wait()
			
			So(int(successCount), ShouldEqual, 100)
			
			// Final counter value should be 100
			finalValue, _ := tree.Find("counter")
			So(finalValue, ShouldEqual, 100)
		})
		
		Convey("Concurrent tree modifications", func() {
			var wg sync.WaitGroup
			
			// Each goroutine creates its own section
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					section := fmt.Sprintf("worker-%d", id)
					
					for j := 0; j < 100; j++ {
						tree.Set(fmt.Sprintf("value-%d", j), section, fmt.Sprintf("item-%d", j))
					}
				}(i)
			}
			
			done := make(chan struct{})
			go func() {
				wg.Wait()
				close(done)
			}()
			
			select {
			case <-done:
				// Success - no deadlock
			case <-time.After(5 * time.Second):
				t.Fatal("Test timed out - possible deadlock")
			}
			
			// Verify all sections exist
			for i := 0; i < 10; i++ {
				section := fmt.Sprintf("worker-%d", i)
				exists := tree.Exists(section)
				So(exists, ShouldBeTrue)
				
				// Check some items
				for j := 0; j < 10; j++ {
					item := fmt.Sprintf("item-%d", j)
					value, err := tree.Find(section, item)
					So(err, ShouldBeNil)
					So(value, ShouldEqual, fmt.Sprintf("value-%d", j))
				}
			}
		})
	})
}

func TestSafeTreeTransactions(t *testing.T) {
	Convey("SafeTree transaction operations", t, func() {
		tree := NewSafeTree(nil)
		
		Convey("Successful transaction", func() {
			tree.Set("initial", "key1")
			tree.Set("initial", "key2")
			
			err := tree.Transaction(func(tx TreeTransaction) error {
				// Read current values
				val1, err := tx.Get("key1")
				if err != nil {
					return err
				}
				
				val2, err := tx.Get("key2")
				if err != nil {
					return err
				}
				
				// Modify values
				tx.Set(fmt.Sprintf("modified-%v", val1), "key1")
				tx.Set(fmt.Sprintf("modified-%v", val2), "key2")
				tx.Set("new-value", "key3")
				
				return nil
			})
			
			So(err, ShouldBeNil)
			
			// Verify changes were applied
			val1, _ := tree.Find("key1")
			So(val1, ShouldEqual, "modified-initial")
			
			val2, _ := tree.Find("key2")
			So(val2, ShouldEqual, "modified-initial")
			
			val3, _ := tree.Find("key3")
			So(val3, ShouldEqual, "new-value")
		})
		
		Convey("Failed transaction rollback", func() {
			tree.Set("original", "key")
			
			err := tree.Transaction(func(tx TreeTransaction) error {
				tx.Set("modified", "key")
				return fmt.Errorf("transaction failed")
			})
			
			So(err, ShouldNotBeNil)
			
			// Original value should remain
			value, _ := tree.Find("key")
			So(value, ShouldEqual, "original")
		})
		
		Convey("Transaction with deletes", func() {
			tree.Set("value1", "key1")
			tree.Set("value2", "key2")
			
			err := tree.Transaction(func(tx TreeTransaction) error {
				tx.Delete("key1")
				tx.Set("new-value", "key3")
				return nil
			})
			
			So(err, ShouldBeNil)
			
			// key1 should be deleted
			So(tree.Exists("key1"), ShouldBeFalse)
			
			// key2 should remain
			value, _ := tree.Find("key2")
			So(value, ShouldEqual, "value2")
			
			// key3 should be created
			value, _ = tree.Find("key3")
			So(value, ShouldEqual, "new-value")
		})
	})
}

func TestSafeTreeReplace(t *testing.T) {
	Convey("SafeTree Replace operation", t, func() {
		tree := NewSafeTree(nil)
		
		Convey("Replace with new data", func() {
			// Set initial data
			tree.Set("original", "key1")
			
			// Replace with new structure
			newData := map[string]interface{}{
				"meta": map[string]interface{}{
					"name": "test",
					"version": 1,
				},
				"data": "new-value",
			}
			
			err := tree.Replace(newData)
			So(err, ShouldBeNil)
			
			// Check merged data
			name, err := tree.Find("meta", "name")
			So(err, ShouldBeNil)
			So(name, ShouldEqual, "test")
			
			version, err := tree.Find("meta", "version")
			So(err, ShouldBeNil)
			So(version, ShouldEqual, 1)
			
			data, err := tree.Find("data")
			So(err, ShouldBeNil)
			So(data, ShouldEqual, "new-value")
			
			// Original key should still exist (merged, not replaced)
			original, err := tree.Find("key1")
			So(err, ShouldBeNil)
			So(original, ShouldEqual, "original")
		})
	})
}

// Benchmark tests for performance comparison
func BenchmarkSafeTreeOperations(b *testing.B) {
	tree := NewSafeTree(nil)
	
	// Initialize some data
	for i := 0; i < 1000; i++ {
		tree.Set(fmt.Sprintf("value-%d", i), "section", fmt.Sprintf("key%d", i))
	}
	
	b.Run("Find", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("key%d", i%1000)
				tree.Find("section", key)
				i++
			}
		})
	})
	
	b.Run("Set", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("newkey%d", i)
				tree.Set(fmt.Sprintf("value-%d", i), "newsection", key)
				i++
			}
		})
	})
	
	b.Run("CompareAndSwap", func(b *testing.B) {
		tree.Set(0, "counter")
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
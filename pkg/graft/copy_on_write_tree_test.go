package graft

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
	
	. "github.com/smartystreets/goconvey/convey"
)

// testTreeWrapper wraps COWTree to provide the test API
type testTreeWrapper struct {
	*COWTree
}

// Set with value first, then path (test API)
func (w *testTreeWrapper) Set(value interface{}, path ...string) error {
	if len(path) == 1 {
		return w.COWTree.Set(path[0], value)
	}
	return w.COWTree.SetAtPath(value, path...)
}

// Get with variadic path (test API)
func (w *testTreeWrapper) Get(path ...string) (interface{}, error) {
	if len(path) == 1 {
		return w.COWTree.Get(path[0])
	}
	// Join multiple path elements with dots
	return w.COWTree.Get(strings.Join(path, "."))
}

// Copy returns a testTreeWrapper
func (w *testTreeWrapper) Copy() *testTreeWrapper {
	return &testTreeWrapper{COWTree: w.COWTree.Copy().(*COWTree)}
}

func newTestTree(init map[interface{}]interface{}) *testTreeWrapper {
	return &testTreeWrapper{COWTree: NewCOWTree(init)}
}

func TestCOWTree(t *testing.T) {
	Convey("COWTree basic operations", t, func() {
		tree := newTestTree(nil)
		
		Convey("Set and Get operations", func() {
			err := tree.Set("test-value", "meta.name")
			So(err, ShouldBeNil)
			
			value, err := tree.Get("meta", "name")
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
		
		Convey("Copy operation (COW semantics)", func() {
			tree.Set("original", "key")
			copied := tree.Copy()
			
			// Modify original
			tree.Set("modified", "key")
			
			// Copy should remain unchanged
			value, err := copied.Get("key")
			So(err, ShouldBeNil)
			So(value, ShouldEqual, "original")
			
			// Original should be modified
			value, err = tree.Get("key")
			So(err, ShouldBeNil)
			So(value, ShouldEqual, "modified")
		})
		
		Convey("Version tracking", func() {
			initialVersion := tree.GetVersion()
			
			tree.Set("key", "value")
			newVersion := tree.GetVersion()
			
			So(newVersion, ShouldBeGreaterThan, initialVersion)
		})
	})
}

func TestCOWTreeCopyOnWrite(t *testing.T) {
	Convey("COWTree Copy-on-Write semantics", t, func() {
		originalData := map[interface{}]interface{}{
			"shared": map[interface{}]interface{}{
				"value": "original",
			},
		}
		
		tree1 := newTestTree(originalData)
		
		Convey("Shallow copy shares memory initially", func() {
			tree2 := tree1.Copy()
			
			// Both trees should have the same value
			value1, err := tree1.Get("shared", "value")
			So(err, ShouldBeNil)
			So(value1, ShouldEqual, "original")
			
			value2, err := tree2.Get("shared", "value")
			So(err, ShouldBeNil)
			So(value2, ShouldEqual, "original")
		})
		
		Convey("Write triggers copy", func() {
			tree2 := tree1.Copy()
			
			// Modify tree2
			err := tree2.Set("modified", "shared", "value")
			So(err, ShouldBeNil)
			
			// tree1 should remain unchanged
			value1, err := tree1.Get("shared", "value")
			So(err, ShouldBeNil)
			So(value1, ShouldEqual, "original")
			
			// tree2 should have the new value
			value2, err := tree2.Get("shared", "value")
			So(err, ShouldBeNil)
			So(value2, ShouldEqual, "modified")
		})
		
		Convey("Multiple copies are independent", func() {
			tree2 := tree1.Copy()
			tree3 := tree1.Copy()
			
			// Modify each tree differently
			tree1.Set("tree1", "shared", "value")
			tree2.Set("tree2", "shared", "value")
			tree3.Set("tree3", "shared", "value")
			
			// Each should have its own value
			value1, _ := tree1.Get("shared", "value")
			So(value1, ShouldEqual, "tree1")
			
			value2, _ := tree2.Get("shared", "value")
			So(value2, ShouldEqual, "tree2")
			
			value3, _ := tree3.Get("shared", "value")
			So(value3, ShouldEqual, "tree3")
		})
	})
}

func TestCOWTreeConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency tests in short mode")
	}
	
	Convey("COWTree concurrent operations", t, func() {
		tree := newTestTree(map[interface{}]interface{}{
			"counters": map[interface{}]interface{}{},
		})
		
		Convey("Concurrent reads and writes", func() {
			var wg sync.WaitGroup
			errors := make(chan error, 100)
			
			// Initialize some data
			for i := 0; i < 10; i++ {
				tree.Set(i, "counters", fmt.Sprintf("counter%d", i))
			}
			
			// Readers
			for i := 0; i < 5; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					for j := 0; j < 100; j++ {
						key := fmt.Sprintf("counter%d", j%10)
						value, err := tree.Get("counters", key)
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
						key := fmt.Sprintf("counter%d", j%10)
						value := fmt.Sprintf("writer-%d-%d", id, j)
						err := tree.Set(value, "counters", key)
						if err != nil {
							errors <- fmt.Errorf("writer %d: %v", id, err)
						}
					}
				}(i)
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
		
		Convey("Concurrent copying and modification", func() {
			var wg sync.WaitGroup
			copies := make(chan *testTreeWrapper, 50)
			
			// Set initial data
			tree.Set("initial", "shared", "value")
			
			// Create copies concurrently
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					for j := 0; j < 5; j++ {
						copy := tree.Copy()
						copies <- copy
						
						// Modify the copy
						copy.Set(fmt.Sprintf("copy-%d-%d", id, j), "shared", "value")
					}
				}(i)
			}
			
			// Modify original tree concurrently
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < 20; i++ {
					tree.Set(fmt.Sprintf("original-%d", i), "shared", "value")
					time.Sleep(time.Microsecond)
				}
			}()
			
			wg.Wait()
			close(copies)
			
			// Verify copies are independent
			copyCount := 0
			for copy := range copies {
				copyCount++
				value, err := copy.Get("shared", "value")
				So(err, ShouldBeNil)
				So(value, ShouldNotBeNil)
			}
			
			So(copyCount, ShouldEqual, 50)
		})
	})
}

func TestCOWTreeTransactions(t *testing.T) {
	Convey("COWTree transaction operations", t, func() {
		tree := newTestTree(map[interface{}]interface{}{
			"account": map[interface{}]interface{}{
				"balance": 100,
			},
		})
		
		Convey("Successful transaction", func() {
			err := tree.Transaction(func(tx TreeTransaction) error {
				// Read current balance
				balance, err := tx.Get("account", "balance")
				if err != nil {
					return err
				}
				
				// Perform transaction
				if balanceInt, ok := balance.(int); ok {
					newBalance := balanceInt - 50
					if newBalance < 0 {
						return fmt.Errorf("insufficient funds")
					}
					
					tx.Set(newBalance, "account", "balance")
					tx.Set("debit", "account", "last_operation")
				} else {
					return fmt.Errorf("balance is not an int: %T %v", balance, balance)
				}
				
				return nil
			})
			
			So(err, ShouldBeNil)
			
			// Verify changes were applied
			balance, err := tree.Get("account", "balance")
			So(err, ShouldBeNil)
			So(balance, ShouldEqual, 50)
			
			lastOp, err := tree.Get("account", "last_operation")
			So(err, ShouldBeNil)
			So(lastOp, ShouldEqual, "debit")
		})
		
		Convey("Failed transaction rollback", func() {
			originalBalance, _ := tree.Get("account", "balance")
			
			err := tree.Transaction(func(tx TreeTransaction) error {
				tx.Set(0, "account", "balance")
				return fmt.Errorf("transaction failed")
			})
			
			So(err, ShouldNotBeNil)
			
			// Balance should remain unchanged
			currentBalance, _ := tree.Get("account", "balance")
			So(currentBalance, ShouldEqual, originalBalance)
		})
	})
}

func TestCOWTreeMemorySharing(t *testing.T) {
	Convey("COWTree memory sharing behavior", t, func() {
		// Create a tree with nested structure
		originalData := map[interface{}]interface{}{
			"level1": map[interface{}]interface{}{
				"level2": map[interface{}]interface{}{
					"level3": map[interface{}]interface{}{
						"value": "deep-value",
					},
				},
			},
		}
		
		tree1 := newTestTree(originalData)
		
		Convey("Deep copy shares unmodified paths", func() {
			tree2 := tree1.Copy()
			tree3 := tree1.Copy()
			
			// Modify only tree2
			tree2.Set("modified", "level1", "level2", "level3", "value")
			
			// tree1 and tree3 should still share memory for unmodified parts
			value1, _ := tree1.Get("level1", "level2", "level3", "value")
			value3, _ := tree3.Get("level1", "level2", "level3", "value")
			
			So(value1, ShouldEqual, "deep-value")
			So(value3, ShouldEqual, "deep-value")
			
			value2, _ := tree2.Get("level1", "level2", "level3", "value")
			So(value2, ShouldEqual, "modified")
		})
		
		Convey("Adding new paths doesn't affect shared structure", func() {
			tree2 := tree1.Copy()
			
			// Add new branch to tree2
			tree2.Set("new-value", "new-branch", "key")
			
			// tree1 shouldn't have the new branch
			So(tree1.Exists("new-branch", "key"), ShouldBeFalse)
			
			// tree2 should have both old and new data
			So(tree2.Exists("new-branch", "key"), ShouldBeTrue)
			So(tree2.Exists("level1", "level2", "level3", "value"), ShouldBeTrue)
		})
	})
}

// Benchmark COW tree vs SafeTree performance
// TODO: SafeTree implementation removed - this benchmark compares COWTree with SafeTree
/*
func BenchmarkCOWTreeVsSafeTree(b *testing.B) {
	initialData := map[interface{}]interface{}{
		"data": map[interface{}]interface{}{},
	}
	
	// Populate with test data
	for i := 0; i < 1000; i++ {
		initialData["data"].(map[interface{}]interface{})[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
	}
	
	b.Run("SafeTree-Find", func(b *testing.B) {
		tree := NewSafeTree(initialData)
		b.ResetTimer()
		
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("key%d", i%1000)
				tree.Get("data", key)
				i++
			}
		})
	})
	
	b.Run("COWTree-Find", func(b *testing.B) {
		tree := newTestTree(initialData)
		b.ResetTimer()
		
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("key%d", i%1000)
				tree.Get("data", key)
				i++
			}
		})
	})
	
	b.Run("SafeTree-Set", func(b *testing.B) {
		tree := NewSafeTree(nil)
		b.ResetTimer()
		
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("newkey%d", i)
				tree.Set(fmt.Sprintf("value%d", i), "data", key)
				i++
			}
		})
	})
	
	b.Run("COWTree-Set", func(b *testing.B) {
		tree := newTestTree(nil)
		b.ResetTimer()
		
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("newkey%d", i)
				tree.Set(fmt.Sprintf("value%d", i), "data", key)
				i++
			}
		})
	})
	
	b.Run("SafeTree-Copy", func(b *testing.B) {
		tree := NewSafeTree(initialData)
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			tree.Copy()
		}
	})
	
	b.Run("COWTree-Copy", func(b *testing.B) {
		tree := newTestTree(initialData)
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			tree.Copy()
		}
	})
	
	b.Run("SafeTree-CopyAndModify", func(b *testing.B) {
		tree := NewSafeTree(initialData)
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			copy := tree.Copy()
			copy.Set(i, "modified", "value")
		}
	})
	
	b.Run("COWTree-CopyAndModify", func(b *testing.B) {
		tree := newTestTree(initialData)
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			copy := tree.Copy()
			copy.Set(i, "modified", "value")
		}
	})
}*/

package graft

import (
	"context"
	"fmt"
	"sync"
	"testing"
	
	. "github.com/smartystreets/goconvey/convey"
)

func TestCOWEvaluator(t *testing.T) {
	Convey("COWEvaluator operations", t, func() {
		data := map[interface{}]interface{}{
			"meta": map[interface{}]interface{}{
				"name":  "test",
				"value": 42,
			},
		}
		
		evaluator := NewCOWEvaluator(data)
		
		Convey("Basic operations", func() {
			ctx := context.Background()
			err := evaluator.Evaluate(ctx)
			So(err, ShouldBeNil)
			
			// Test tree access
			tree := evaluator.GetTree()
			So(tree, ShouldNotBeNil)
			
			// Test value operations
			err = evaluator.SetValue("new-value", "meta", "newkey")
			So(err, ShouldBeNil)
			
			value, err := evaluator.GetValue("meta", "newkey")
			So(err, ShouldBeNil)
			So(value, ShouldEqual, "new-value")
		})
		
		Convey("Snapshot creation", func() {
			evaluator.SetValue("original", "snapshot", "value")
			
			// Create snapshot
			snapshot := evaluator.CreateSnapshot()
			So(snapshot, ShouldNotBeNil)
			
			// Modify original
			evaluator.SetValue("modified", "snapshot", "value")
			
			// Snapshot should retain original value
			origValue, err := snapshot.GetValue("snapshot", "value")
			So(err, ShouldBeNil)
			So(origValue, ShouldEqual, "original")
			
			// Original should have new value
			newValue, err := evaluator.GetValue("snapshot", "value")
			So(err, ShouldBeNil)
			So(newValue, ShouldEqual, "modified")
		})
		
		Convey("Version tracking", func() {
			initialVersion := evaluator.GetVersion()
			
			evaluator.SetValue("test", "version", "key")
			newVersion := evaluator.GetVersion()
			
			So(newVersion, ShouldBeGreaterThan, initialVersion)
		})
	})
}

func TestEnhancedMigrationHelper(t *testing.T) {
	Convey("EnhancedMigrationHelper operations", t, func() {
		originalData := map[interface{}]interface{}{
			"meta": map[interface{}]interface{}{
				"name": "original",
			},
		}
		
		helper := NewEnhancedMigrationHelper(originalData)
		
		Convey("Basic helper operations", func() {
			cowEval := helper.GetCOWEvaluator()
			So(cowEval, ShouldNotBeNil)
			
			tree := helper.GetThreadSafeTree()
			So(tree, ShouldNotBeNil)
			
			value, err := tree.Get("meta.name")
			So(err, ShouldBeNil)
			So(value, ShouldEqual, "original")
		})
		
		Convey("Snapshot management", func() {
			// Create multiple snapshots
			snapshot1 := helper.CreateSnapshot()
			
			helper.GetCOWEvaluator().SetValue("modified1", "meta", "name")
			snapshot2 := helper.CreateSnapshot()
			
			helper.GetCOWEvaluator().SetValue("modified2", "meta", "name")
			snapshot3 := helper.CreateSnapshot()
			
			// Get all snapshots
			snapshots := helper.GetSnapshots()
			So(len(snapshots), ShouldEqual, 3)
			
			// Verify each snapshot has correct value
			val1, _ := snapshot1.GetValue("meta", "name")
			So(val1, ShouldEqual, "original")
			
			val2, _ := snapshot2.GetValue("meta", "name")
			So(val2, ShouldEqual, "modified1")
			
			val3, _ := snapshot3.GetValue("meta", "name")
			So(val3, ShouldEqual, "modified2")
		})
		
		Convey("Traditional evaluator integration", func() {
			// Export to traditional evaluator
			traditionalEv, err := helper.ExportToEvaluator()
			So(err, ShouldBeNil)
			So(traditionalEv, ShouldNotBeNil)
			
			// Modify traditional evaluator
			traditionalEv.Tree["meta"].(map[interface{}]interface{})["modified"] = true
			
			// Update helper from traditional evaluator
			err = helper.UpdateFromEvaluator(traditionalEv)
			So(err, ShouldBeNil)
			
			// Verify change was applied
			value, err := helper.GetCOWEvaluator().GetValue("meta", "modified")
			So(err, ShouldBeNil)
			So(value, ShouldEqual, true)
		})
	})
}

func TestCOWEvaluatorConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency tests in short mode")
	}
	
	Convey("COWEvaluator concurrent operations", t, func() {
		data := map[interface{}]interface{}{
			"counters": map[interface{}]interface{}{},
		}
		
		evaluator := NewCOWEvaluator(data)
		
		Convey("Concurrent snapshots and modifications", func() {
			var wg sync.WaitGroup
			snapshots := make(chan *COWEvaluator, 100)
			
			// Create snapshots concurrently
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					
					for j := 0; j < 10; j++ {
						// Create snapshot
						snapshot := evaluator.CreateSnapshot()
						snapshots <- snapshot
						
						// Modify evaluator
						evaluator.SetValue(fmt.Sprintf("worker-%d-%d", id, j), "counters", fmt.Sprintf("key%d", id))
					}
				}(i)
			}
			
			wg.Wait()
			close(snapshots)
			
			// Verify snapshots are independent
			snapshotCount := 0
			for snapshot := range snapshots {
				snapshotCount++
				So(snapshot, ShouldNotBeNil)
				
				// Each snapshot should be functional
				err := snapshot.SetValue("test", "test-key")
				So(err, ShouldBeNil)
			}
			
			So(snapshotCount, ShouldEqual, 100)
		})
		
		Convey("Concurrent evaluator operations", func() {
			var wg sync.WaitGroup
			errors := make(chan error, 50)
			
			// Multiple workers performing operations
			for i := 0; i < 5; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					
					for j := 0; j < 10; j++ {
						key := fmt.Sprintf("worker%d-key%d", id, j)
						value := fmt.Sprintf("value-%d-%d", id, j)
						
						// Set value
						if err := evaluator.SetValue(value, "concurrent", key); err != nil {
							errors <- err
							return
						}
						
						// Get value
						if _, err := evaluator.GetValue("concurrent", key); err != nil {
							errors <- err
							return
						}
						
						// Create snapshot
						snapshot := evaluator.CreateSnapshot()
						if snapshot == nil {
							errors <- fmt.Errorf("nil snapshot from worker %d", id)
							return
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
	})
}

func TestCOWTreeFactory(t *testing.T) {
	Convey("COWTreeFactory operations", t, func() {
		factory := NewCOWTreeFactory()
		
		Convey("Create from data", func() {
			data := map[interface{}]interface{}{
				"test": "value",
			}
			
			tree := factory.CreateFromData(data)
			So(tree, ShouldNotBeNil)
			
			value, err := tree.Get("test")
			So(err, ShouldBeNil)
			So(value, ShouldEqual, "value")
		})
		
		Convey("Create empty", func() {
			tree := factory.CreateEmpty()
			So(tree, ShouldNotBeNil)
			
			// COWTree's Set method takes path first, then value
			cowTree := tree.(*COWTree)
			err := cowTree.Set("new-key", "new-value")
			So(err, ShouldBeNil)
			
			value, err := tree.Get("new-key")
			So(err, ShouldBeNil)
			So(value, ShouldEqual, "new-value")
		})
		
		Convey("Create from YAML", func() {
			tree, err := factory.CreateFromYAML([]byte("test: value"))
			So(err, ShouldBeNil)
			So(tree, ShouldNotBeNil)
		})
	})
}

func TestCOWPerformanceMonitor(t *testing.T) {
	Convey("COWPerformanceMonitor operations", t, func() {
		monitor := NewCOWPerformanceMonitor()
		
		Convey("Record operations", func() {
			monitor.RecordCopy()
			monitor.RecordCopy()
			monitor.RecordModify()
			
			stats := monitor.GetStats()
			So(stats["copies"], ShouldEqual, 2)
			So(stats["modifies"], ShouldEqual, 1)
		})
		
		Convey("Concurrent recording", func() {
			var wg sync.WaitGroup
			
			// Multiple goroutines recording operations
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < 100; j++ {
						monitor.RecordCopy()
						monitor.RecordModify()
					}
				}()
			}
			
			wg.Wait()
			
			stats := monitor.GetStats()
			So(stats["copies"], ShouldEqual, 1000)
			So(stats["modifies"], ShouldEqual, 1000)
		})
	})
}

func TestCOWTreeComparator(t *testing.T) {
	Convey("COWTreeComparator operations", t, func() {
		comparator := NewCOWTreeComparator()
		
		Convey("Compare trees", func() {
			data := map[interface{}]interface{}{
				"test": "value",
			}
			
			tree1 := NewCOWTree(data)
			tree2 := tree1.Copy()
			
			// Initially same
			diff, err := comparator.Compare(tree1, tree2)
			So(err, ShouldBeNil)
			So(diff["different"], ShouldBeFalse)
			
			// Modify one tree
			tree1.Set("modified", "test")
			
			// Now different
			diff, err = comparator.Compare(tree1, tree2)
			So(err, ShouldBeNil)
			So(diff["different"], ShouldBeTrue)
		})
	})
}

// Benchmark COW evaluator operations
func BenchmarkCOWEvaluator(b *testing.B) {
	data := map[interface{}]interface{}{
		"data": map[interface{}]interface{}{},
	}
	
	// Populate with test data
	for i := 0; i < 1000; i++ {
		data["data"].(map[interface{}]interface{})[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
	}
	
	evaluator := NewCOWEvaluator(data)
	ctx := context.Background()
	
	b.Run("Evaluate", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			evaluator.Evaluate(ctx)
		}
	})
	
	b.Run("SetValue", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("newkey%d", i)
				evaluator.SetValue(fmt.Sprintf("value%d", i), "data", key)
				i++
			}
		})
	})
	
	b.Run("GetValue", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("key%d", i%1000)
				evaluator.GetValue("data", key)
				i++
			}
		})
	})
	
	b.Run("CreateSnapshot", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			evaluator.CreateSnapshot()
		}
	})
	
	b.Run("SnapshotAndModify", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			snapshot := evaluator.CreateSnapshot()
			snapshot.SetValue(i, "snapshot", "value")
		}
	})
}
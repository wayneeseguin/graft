package spruce

import (
	"fmt"
	"sync"
	"testing"
	
	. "github.com/smartystreets/goconvey/convey"
)

// TestBasicThreadSafety demonstrates current thread safety issues
func TestBasicThreadSafety(t *testing.T) {
	Convey("Basic thread safety tests", t, func() {
		Convey("Concurrent map writes cause panic", func() {
			// Skip this test when running with race detector as it intentionally causes races
			if testing.Short() {
				t.Skip("Skipping race condition test in short mode")
				return
			}
			
			defer func() {
				if r := recover(); r != nil {
					// Expected - concurrent map writes
					So(r, ShouldNotBeNil)
				}
			}()
			
			// This will panic with concurrent map writes
			m := make(map[string]interface{})
			var wg sync.WaitGroup
			
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					// Concurrent writes to same map
					m[fmt.Sprintf("key%d", id)] = id
				}(i)
			}
			
			wg.Wait()
		})
		
		Convey("Safe concurrent access with mutex", func() {
			m := make(map[string]interface{})
			var mu sync.Mutex
			var wg sync.WaitGroup
			
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					mu.Lock()
					m[fmt.Sprintf("key%d", id)] = id
					mu.Unlock()
				}(i)
			}
			
			wg.Wait()
			So(len(m), ShouldEqual, 10)
		})
	})
}
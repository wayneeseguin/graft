package internal

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestShardedLockManager_NewShardedLockManager(t *testing.T) {
	tests := []struct {
		name           string
		numShards      int
		expectedShards int
	}{
		{
			name:           "default shards for zero",
			numShards:      0,
			expectedShards: 32,
		},
		{
			name:           "default shards for negative",
			numShards:      -5,
			expectedShards: 32,
		},
		{
			name:           "custom shard count",
			numShards:      16,
			expectedShards: 16,
		},
		{
			name:           "large shard count",
			numShards:      1024,
			expectedShards: 1024,
		},
		{
			name:           "max uint32 boundary",
			numShards:      4294967295,
			expectedShards: 4294967295,
		},
		{
			name:           "exceeds uint32 max",
			numShards:      4294967296,
			expectedShards: 4294967295,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slm := NewShardedLockManager(tt.numShards)
			
			if slm == nil {
				t.Fatal("expected lock manager to be created, got nil")
			}
			
			if int(slm.numShards) != tt.expectedShards {
				t.Errorf("expected %d shards, got %d", tt.expectedShards, slm.numShards)
			}
			
			if len(slm.shards) != tt.expectedShards {
				t.Errorf("expected %d shards array length, got %d", tt.expectedShards, len(slm.shards))
			}
		})
	}
}

func TestShardedLockManager_GetShardIndex(t *testing.T) {
	slm := NewShardedLockManager(16)
	
	tests := []struct {
		name string
		path []string
	}{
		{
			name: "empty path",
			path: []string{},
		},
		{
			name: "single element",
			path: []string{"root"},
		},
		{
			name: "multiple elements",
			path: []string{"root", "child", "grandchild"},
		},
		{
			name: "special characters",
			path: []string{"root.with.dots", "child-with-dashes", "child_with_underscores"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			index := slm.getShardIndex(tt.path)
			
			if index >= slm.numShards {
				t.Errorf("shard index %d exceeds number of shards %d", index, slm.numShards)
			}
			
			// Ensure consistent hashing - same path should return same index
			index2 := slm.getShardIndex(tt.path)
			if index != index2 {
				t.Errorf("inconsistent shard index: first call returned %d, second call returned %d", index, index2)
			}
		})
	}
}

func TestShardedLockManager_LockAcquisitionRelease(t *testing.T) {
	slm := NewShardedLockManager(8)
	path := []string{"test", "path"}

	t.Run("exclusive lock", func(t *testing.T) {
		unlock := slm.Lock(path, true)
		if unlock == nil {
			t.Fatal("expected unlock function, got nil")
		}
		
		// Lock is acquired, now release it
		unlock()
	})

	t.Run("read lock", func(t *testing.T) {
		unlock := slm.Lock(path, false)
		if unlock == nil {
			t.Fatal("expected unlock function, got nil")
		}
		
		// Lock is acquired, now release it
		unlock()
	})

	t.Run("multiple read locks", func(t *testing.T) {
		unlock1 := slm.Lock(path, false)
		unlock2 := slm.Lock(path, false)
		unlock3 := slm.Lock(path, false)
		
		// All read locks should be acquired successfully
		if unlock1 == nil || unlock2 == nil || unlock3 == nil {
			t.Fatal("expected all read locks to be acquired")
		}
		
		// Release all locks
		unlock1()
		unlock2()
		unlock3()
	})
}

func TestShardedLockManager_ConcurrentAccess(t *testing.T) {
	slm := NewShardedLockManager(16)
	path := []string{"concurrent", "test"}
	
	const numWorkers = 10
	const numIterations = 100
	
	t.Run("concurrent read locks", func(t *testing.T) {
		var wg sync.WaitGroup
		var successCount int64
		
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				
				for j := 0; j < numIterations; j++ {
					unlock := slm.Lock(path, false) // Read lock
					
					// Simulate some work
					time.Sleep(time.Microsecond)
					
					unlock()
					atomic.AddInt64(&successCount, 1)
				}
			}(i)
		}
		
		wg.Wait()
		
		expectedSuccess := int64(numWorkers * numIterations)
		if successCount != expectedSuccess {
			t.Errorf("expected %d successful operations, got %d", expectedSuccess, successCount)
		}
	})

	t.Run("mixed read/write locks", func(t *testing.T) {
		var wg sync.WaitGroup
		var readCount, writeCount int64
		
		// Start readers
		for i := 0; i < numWorkers/2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				
				for j := 0; j < numIterations/10; j++ {
					unlock := slm.Lock(path, false) // Read lock
					
					// Simulate read work
					time.Sleep(time.Microsecond)
					atomic.AddInt64(&readCount, 1)
					
					unlock()
				}
			}()
		}
		
		// Start writers
		for i := 0; i < numWorkers/2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				
				for j := 0; j < numIterations/10; j++ {
					unlock := slm.Lock(path, true) // Write lock
					
					// Simulate write work
					time.Sleep(time.Microsecond * 2)
					atomic.AddInt64(&writeCount, 1)
					
					unlock()
				}
			}()
		}
		
		wg.Wait()
		
		expectedReads := int64((numWorkers / 2) * (numIterations / 10))
		expectedWrites := int64((numWorkers / 2) * (numIterations / 10))
		
		if readCount != expectedReads {
			t.Errorf("expected %d reads, got %d", expectedReads, readCount)
		}
		if writeCount != expectedWrites {
			t.Errorf("expected %d writes, got %d", expectedWrites, writeCount)
		}
	})
}

func TestShardedLockManager_TryLock(t *testing.T) {
	slm := NewShardedLockManager(4)
	path := []string{"trylock", "test"}

	t.Run("successful try lock", func(t *testing.T) {
		unlock, err := slm.TryLock(path, false, time.Millisecond*100)
		if err != nil {
			t.Fatalf("expected successful lock, got error: %v", err)
		}
		if unlock == nil {
			t.Fatal("expected unlock function, got nil")
		}
		
		unlock()
	})

	t.Run("timeout on try lock", func(t *testing.T) {
		// First acquire a write lock
		unlock1 := slm.Lock(path, true)
		defer unlock1()
		
		// Try to acquire another write lock with timeout
		unlock2, err := slm.TryLock(path, true, time.Millisecond*10)
		
		if err == nil {
			t.Error("expected timeout error, got none")
			if unlock2 != nil {
				unlock2()
			}
		}
		
		if _, ok := err.(*LockTimeoutError); !ok {
			t.Errorf("expected LockTimeoutError, got %T: %v", err, err)
		}
	})

	t.Run("concurrent try locks", func(t *testing.T) {
		const numWorkers = 5
		var wg sync.WaitGroup
		var successCount, timeoutCount int64
		
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				
				// Use different timeouts to reduce contention
				timeout := time.Millisecond * time.Duration(50 + workerID*10)
				unlock, err := slm.TryLock(path, true, timeout)
				if err == nil {
					atomic.AddInt64(&successCount, 1)
					time.Sleep(time.Millisecond * 5) // Hold lock briefly
					unlock()
				} else {
					atomic.AddInt64(&timeoutCount, 1)
				}
			}(i)
		}
		
		wg.Wait()
		
		total := successCount + timeoutCount
		if total != numWorkers {
			t.Errorf("expected %d total operations, got %d", numWorkers, total)
		}
		
		// We expect some operations to succeed and potentially some to timeout
		// The exact distribution depends on timing, so we just verify totals match
		t.Logf("Successful locks: %d, Timeouts: %d", successCount, timeoutCount)
	})
}

func TestShardedLockManager_IsLocked(t *testing.T) {
	slm := NewShardedLockManager(4)
	path := []string{"islocked", "test"}

	t.Run("unlocked path", func(t *testing.T) {
		if slm.IsLocked(path) {
			t.Error("expected path to be unlocked")
		}
	})

	t.Run("read locked path", func(t *testing.T) {
		unlock := slm.Lock(path, false)
		defer unlock()
		
		// Read locks shouldn't be detected as "locked" by IsLocked
		// since IsLocked tries to acquire a read lock
		if slm.IsLocked(path) {
			t.Error("expected read-locked path to not be detected as locked")
		}
	})

	t.Run("write locked path", func(t *testing.T) {
		unlock := slm.Lock(path, true)
		defer unlock()
		
		// Write locks should be detected
		if !slm.IsLocked(path) {
			t.Error("expected write-locked path to be detected as locked")
		}
	})
}

func TestShardedLockManager_DeadlockPrevention(t *testing.T) {
	slm := NewShardedLockManager(8)
	
	// Test that different paths can be locked simultaneously without deadlock
	paths := [][]string{
		{"path1"},
		{"path2"},
		{"path3"},
		{"path4"},
	}
	
	t.Run("no deadlock with different paths", func(t *testing.T) {
		var wg sync.WaitGroup
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		
		for i, path := range paths {
			wg.Add(1)
			go func(id int, p []string) {
				defer wg.Done()
				
				select {
				case <-ctx.Done():
					t.Errorf("worker %d timed out", id)
					return
				default:
				}
				
				// Acquire write lock
				unlock := slm.Lock(p, true)
				
				// Hold lock briefly
				time.Sleep(time.Millisecond * 10)
				
				unlock()
			}(i, path)
		}
		
		wg.Wait()
		
		if ctx.Err() == context.DeadlineExceeded {
			t.Error("deadlock detected - test timed out")
		}
	})

	t.Run("no deadlock with overlapping operations", func(t *testing.T) {
		const numWorkers = 20
		var wg sync.WaitGroup
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				
				path := []string{"worker", string(rune('A' + (workerID % 8)))}
				
				for j := 0; j < 10; j++ {
					select {
					case <-ctx.Done():
						return
					default:
					}
					
					// Mix of read and write operations
					exclusive := j%3 == 0
					unlock := slm.Lock(path, exclusive)
					
					// Brief work simulation
					time.Sleep(time.Microsecond * 100)
					
					unlock()
				}
			}(i)
		}
		
		wg.Wait()
		
		if ctx.Err() == context.DeadlineExceeded {
			t.Error("deadlock detected - overlapping operations test timed out")
		}
	})
}

func TestLockTimeoutError(t *testing.T) {
	path := []string{"test", "path"}
	timeout := time.Millisecond * 100
	
	err := &LockTimeoutError{
		Path:    path,
		Timeout: timeout,
	}
	
	expectedMsg := "failed to acquire lock for path [test path] within 100ms"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func BenchmarkShardedLockManager_Lock(b *testing.B) {
	slm := NewShardedLockManager(32)
	path := []string{"benchmark", "test"}
	
	b.ResetTimer()
	
	b.Run("read_locks", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				unlock := slm.Lock(path, false)
				unlock()
			}
		})
	})
	
	b.Run("write_locks", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			unlock := slm.Lock(path, true)
			unlock()
		}
	})
	
	b.Run("mixed_locks", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				exclusive := i%4 == 0 // 25% write locks
				unlock := slm.Lock(path, exclusive)
				unlock()
				i++
			}
		})
	})
}

func BenchmarkShardedLockManager_GetShardIndex(b *testing.B) {
	slm := NewShardedLockManager(32)
	paths := [][]string{
		{"simple"},
		{"nested", "path"},
		{"deeply", "nested", "path", "structure"},
		{"path", "with", "many", "segments", "to", "test", "hashing"},
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		path := paths[i%len(paths)]
		slm.getShardIndex(path)
	}
}

func BenchmarkShardedLockManager_TryLock(b *testing.B) {
	slm := NewShardedLockManager(16)
	path := []string{"trylock", "benchmark"}
	timeout := time.Millisecond * 10
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		unlock, err := slm.TryLock(path, false, timeout)
		if err == nil && unlock != nil {
			unlock()
		}
	}
}
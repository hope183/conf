package conf

import (
	"fmt"
	"sync"
	"testing"
)

func TestConcurrentAccess(t *testing.T) {
	storage := newMockStorage()
	manager := NewSettingManager(storage)

	const (
		numGoroutines = 10
		numOperations = 100
	)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// 用于收集错误
	errChan := make(chan error, numGoroutines*numOperations)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)

				// 写入
				err := manager.Set(key, j)
				if err != nil {
					errChan <- fmt.Errorf("set error: %w", err)
					continue
				}

				// 读取 - 使用类型化的 Get
				value, err := Get[int](key)
				if err != nil {
					errChan <- fmt.Errorf("get error: %w", err)
					continue
				}
				if *value != j {
					errChan <- fmt.Errorf("value mismatch for key %s: got %v, want %v", key, *value, j)
					continue
				}

				// 删除
				err = manager.Delete(key)
				if err != nil {
					errChan <- fmt.Errorf("delete error: %w", err)
				}
			}
		}(i)
	}

	// 等待所有 goroutine 完成
	wg.Wait()
	close(errChan)

	// 收集所有错误
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	// 如果有错误，报告它们
	if len(errors) > 0 {
		for _, err := range errors {
			t.Errorf("concurrent test error: %v", err)
		}
		t.Fatalf("concurrent test failed with %d errors", len(errors))
	}
}

// 添加基准测试
func BenchmarkConcurrentAccess(b *testing.B) {
	storage := newMockStorage()
	manager := NewSettingManager(storage)

	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			key := fmt.Sprintf("bench-key-%d", counter)

			// 写入
			err := manager.Set(key, counter)
			if err != nil {
				b.Fatal(err)
			}

			// 读取
			value, err := Get[int](key)
			if err != nil {
				b.Fatal(err)
			}
			if *value != counter {
				b.Fatalf("value mismatch: got %v, want %v", *value, counter)
			}

			// 删除
			err = manager.Delete(key)
			if err != nil {
				b.Fatal(err)
			}

			counter++
		}
	})
}

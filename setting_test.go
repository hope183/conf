package conf

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 模拟存储实现
type mockStorage struct {
	sync.RWMutex
	data map[string]string
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		data: make(map[string]string),
	}
}

func (ms *mockStorage) Get(key string) (string, error) {
	ms.RLock()
	defer ms.RUnlock()

	if value, ok := ms.data[key]; ok {
		return value, nil
	}
	return "", ErrKeyNotFound
}

func (ms *mockStorage) Set(key, value string) error {
	ms.Lock()
	defer ms.Unlock()

	ms.data[key] = value
	return nil
}

func (ms *mockStorage) Delete(key string) error {
	ms.Lock()
	defer ms.Unlock()

	delete(ms.data, key)
	return nil
}

func TestSettingManager_BasicOperations(t *testing.T) {
	storage := newMockStorage()
	manager := NewSettingManager(storage)

	t.Run("string operations", func(t *testing.T) {
		// 测试字符串类型
		err := manager.Set("test.string", "hello world")
		require.NoError(t, err)

		// 直接获取
		rawValue, err := manager.Get("test.string")
		require.NoError(t, err)
		assert.Equal(t, "hello world", rawValue)

		// 类型化获取
		typedValue, err := Get[string]("test.string")
		require.NoError(t, err)
		assert.Equal(t, "hello world", *typedValue)
	})

	t.Run("int operations", func(t *testing.T) {
		// 测试整数类型
		err := manager.Set("test.int", 42)
		require.NoError(t, err)

		// 直接获取会得到字符串
		rawValue, err := manager.Get("test.int")
		require.NoError(t, err)
		assert.Equal(t, "42", rawValue)

		// 类型化获取会得到正确的类型
		typedValue, err := Get[int]("test.int")
		require.NoError(t, err)
		assert.Equal(t, 42, *typedValue)
	})

	t.Run("bool operations", func(t *testing.T) {
		// 测试布尔类型
		err := manager.Set("test.bool", true)
		require.NoError(t, err)

		// 直接获取会得到字符串
		rawValue, err := manager.Get("test.bool")
		require.NoError(t, err)
		assert.Equal(t, "true", rawValue)

		// 类型化获取会得到正确的类型
		typedValue, err := Get[bool]("test.bool")
		require.NoError(t, err)
		assert.Equal(t, true, *typedValue)
	})

	t.Run("float operations", func(t *testing.T) {
		// 测试浮点类型
		err := manager.Set("test.float", 3.14)
		require.NoError(t, err)

		// 直接获取会得到字符串
		rawValue, err := manager.Get("test.float")
		require.NoError(t, err)
		assert.Equal(t, "3.14", rawValue)

		// 类型化获取会得到正确的类型
		typedValue, err := Get[float64]("test.float")
		require.NoError(t, err)
		assert.Equal(t, 3.14, *typedValue)
	})

	t.Run("complex type operations", func(t *testing.T) {
		type Config struct {
			Host string `json:"host"`
			Port int    `json:"port"`
		}

		cfg := Config{
			Host: "localhost",
			Port: 8080,
		}

		// 测试复杂类型
		err := manager.Set("test.config", cfg)
		require.NoError(t, err)

		// 直接获取会得到 JSON 字符串
		rawValue, err := manager.Get("test.config")
		require.NoError(t, err)
		assert.Contains(t, rawValue.(string), "localhost")
		assert.Contains(t, rawValue.(string), "8080")

		// 类型化获取会得到正确的类型
		typedValue, err := Get[Config]("test.config")
		require.NoError(t, err)
		assert.Equal(t, cfg, *typedValue)
	})

	t.Run("error cases", func(t *testing.T) {
		// 测试获取不存在的键
		_, err := manager.Get("not.exist")
		assert.Error(t, err)

		// 测试类型转换错误
		err = manager.Set("test.number", "not a number")
		require.NoError(t, err)
		_, err = Get[int]("test.number")
		assert.Error(t, err)
	})
}

// 测试缓存机制
func TestSettingManager_Cache(t *testing.T) {
	storage := newMockStorage()
	manager := NewSettingManager(storage)

	// 设置值
	err := manager.Set("test.key", "value")
	require.NoError(t, err)

	// 第一次获取，应该从存储层读取
	value1, err := manager.Get("test.key")
	require.NoError(t, err)
	assert.Equal(t, "value", value1)

	// 修改存储层的值（模拟其他实例的修改）
	storage.data["test.key"] = "modified"

	// 第二次获取，应该从缓存读取
	value2, err := manager.Get("test.key")
	require.NoError(t, err)
	assert.Equal(t, "value", value2) // 仍然是缓存的值

	// 删除键，应该同时清除缓存
	err = manager.Delete("test.key")
	require.NoError(t, err)

	// 再次获取，应该报错
	_, err = manager.Get("test.key")
	assert.Error(t, err)
}

// 测试并发安全性
func TestSettingManager_Concurrent(t *testing.T) {
	storage := newMockStorage()
	manager := NewSettingManager(storage)

	const (
		goroutines = 5  // 减少并发数
		operations = 50 // 减少操作数
	)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	// 用于收集错误
	errChan := make(chan error, goroutines*operations)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				key := fmt.Sprintf("test.key.%d.%d", id, j)

				// 设置值
				if err := manager.Set(key, j); err != nil {
					errChan <- fmt.Errorf("set error: %w", err)
					continue
				}

				// 获取值
				value, err := Get[int](key)
				if err != nil {
					errChan <- fmt.Errorf("get error: %w", err)
					continue
				}
				if *value != j {
					errChan <- fmt.Errorf("value mismatch: got %v, want %v", *value, j)
					continue
				}

				// 删除值
				if err := manager.Delete(key); err != nil {
					errChan <- fmt.Errorf("delete error: %w", err)
				}
			}
		}(i)
	}

	// 等待所有 goroutine 完成
	wg.Wait()
	close(errChan)

	// 检查是否有错误
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		for _, err := range errors {
			t.Errorf("concurrent test error: %v", err)
		}
		t.Fatalf("concurrent test failed with %d errors", len(errors))
	}
}

func TestSettingOperations(t *testing.T) {
	// 假设这里有一个设置测试数据库的函数
	setupTestDB(t)

	t.Run("Set and Get String", func(t *testing.T) {
		err := Set("app_name", "My Awesome App")
		if err != nil {
			t.Fatalf("Error setting string: %v", err)
		}

		value, err := Get[string]("app_name")
		if err != nil {
			t.Fatalf("Error getting string: %v", err)
		}
		if *value != "My Awesome App" {
			t.Errorf("Expected 'My Awesome App', got '%s'", *value)
		}
	})

	t.Run("Set and Get Integer", func(t *testing.T) {
		err := Set("max_connections", 100)
		if err != nil {
			t.Fatalf("Error setting int: %v", err)
		}

		value, err := Get[int]("max_connections")
		if err != nil {
			t.Fatalf("Error getting int: %v", err)
		}
		if *value != 100 {
			t.Errorf("Expected 100, got %d", *value)
		}
	})

	t.Run("Set and Get Float", func(t *testing.T) {
		err := Set("pi_value", 3.14159)
		if err != nil {
			t.Fatalf("Error setting float: %v", err)
		}

		value, err := Get[float64]("pi_value")
		if err != nil {
			t.Fatalf("Error getting float: %v", err)
		}
		if *value != 3.14159 {
			t.Errorf("Expected 3.14159, got %f", *value)
		}
	})

	t.Run("Set and Get Boolean", func(t *testing.T) {
		err := Set("debug_mode", true)
		if err != nil {
			t.Fatalf("Error setting bool: %v", err)
		}

		value, err := Get[bool]("debug_mode")
		if err != nil {
			t.Fatalf("Error getting bool: %v", err)
		}
		if *value != true {
			t.Errorf("Expected true, got %v", *value)
		}
	})

	t.Run("Set and Get Struct", func(t *testing.T) {
		type Config struct {
			Host string
			Port int
		}
		config := Config{Host: "localhost", Port: 5432}
		err := Set("database_config", config)
		if err != nil {
			t.Fatalf("Error setting struct: %v", err)
		}

		value, err := Get[Config]("database_config")
		if err != nil {
			t.Fatalf("Error getting struct: %v", err)
		}
		if !reflect.DeepEqual(*value, config) {
			t.Errorf("Expected %+v, got %+v", config, *value)
		}
	})

	t.Run("Delete Setting", func(t *testing.T) {
		err := Delete("debug_mode")
		if err != nil {
			t.Fatalf("Error deleting setting: %v", err)
		}

		_, err = Get[bool]("debug_mode")
		if err == nil {
			t.Error("Expected error when getting deleted setting, got nil")
		}
	})
}

var testDB *gorm.DB

func setupTestDB(t *testing.T) {
	t.Helper()

	var err error
	testDB, err = gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("Error opening test database: %v", err)
	}

	// Create the settings table
	testDB.Exec(`
		CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Error creating settings table: %v", err)
	}

}

func BenchmarkSettingManager_Get(b *testing.B) {
	storage := newMockStorage()
	manager := NewSettingManager(storage)

	err := manager.Set("bench.key", "value")
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.Get("bench.key")
	}
}

func FuzzSettingManager_Set(f *testing.F) {
	storage := newMockStorage()
	manager := NewSettingManager(storage)

	f.Add("test.key", "value")
	f.Fuzz(func(t *testing.T, key string, value string) {
		err := manager.Set(key, value)
		if err != nil {
			t.Skip()
		}

		got, err := manager.Get(key)
		if err != nil {
			t.Error(err)
		}
		if got != value {
			t.Errorf("got %v, want %v", got, value)
		}
	})
}

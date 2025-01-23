package conf

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSettingCache(t *testing.T) {
	t.Run("basic operations", func(t *testing.T) {
		cache := &settingCache{
			cache:      make(map[string]string),
			lastAccess: make(map[string]time.Time),
			maxSize:    2,
			expiration: time.Second,
		}

		// 设置值
		cache.Set("key1", "value1")

		// 获取值
		value, ok := cache.Get("key1")
		assert.True(t, ok)
		assert.Equal(t, "value1", value)

		// 删除值
		cache.Delete("key1")
		_, ok = cache.Get("key1")
		assert.False(t, ok)
	})

	t.Run("max size limit", func(t *testing.T) {
		cache := &settingCache{
			cache:      make(map[string]string),
			lastAccess: make(map[string]time.Time),
			maxSize:    2,
			expiration: time.Second,
		}

		// 按顺序设置值，并记录访问时间
		cache.Set("key1", "value1")
		time.Sleep(10 * time.Millisecond)

		cache.Set("key2", "value2")
		time.Sleep(10 * time.Millisecond)

		// key3 的设置应该触发清理最早的 key1
		cache.Set("key3", "value3")

		// 验证 key1 已被清理
		_, ok := cache.Get("key1")
		assert.False(t, ok, "key1 should have been evicted")

		// 验证 key2 和 key3 存在
		v2, ok := cache.Get("key2")
		assert.True(t, ok, "key2 should exist")
		assert.Equal(t, "value2", v2)

		v3, ok := cache.Get("key3")
		assert.True(t, ok, "key3 should exist")
		assert.Equal(t, "value3", v3)
	})

	t.Run("expiration", func(t *testing.T) {
		cache := &settingCache{
			cache:      make(map[string]string),
			lastAccess: make(map[string]time.Time),
			maxSize:    10,
			expiration: 100 * time.Millisecond,
		}

		cache.Set("key", "value")

		// 立即获取应该成功
		value, ok := cache.Get("key")
		assert.True(t, ok)
		assert.Equal(t, "value", value)

		// 等待过期（等待时间要足够长）
		time.Sleep(150 * time.Millisecond)

		// 获取应该失败
		_, ok = cache.Get("key")
		assert.False(t, ok, "key should have expired")
	})

	t.Run("concurrent access", func(t *testing.T) {
		cache := &settingCache{
			cache:      make(map[string]string),
			lastAccess: make(map[string]time.Time),
			maxSize:    100,
			expiration: time.Second,
		}

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				key := fmt.Sprintf("key%d", id)
				value := fmt.Sprintf("value%d", id)

				cache.Set(key, value)
				time.Sleep(10 * time.Millisecond)

				got, ok := cache.Get(key)
				assert.True(t, ok)
				assert.Equal(t, value, got)

				cache.Delete(key)
			}(i)
		}
		wg.Wait()
	})
}

func TestSettingCache_Cleanup(t *testing.T) {
	cache := &settingCache{
		cache:      make(map[string]string),
		lastAccess: make(map[string]time.Time),
		maxSize:    2,
		expiration: 100 * time.Millisecond,
	}

	// 填充缓存
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key%d", i)
		cache.Set(key, fmt.Sprintf("value%d", i))
		time.Sleep(10 * time.Millisecond)
	}

	// 验证缓存大小
	assert.LessOrEqual(t, len(cache.cache), cache.maxSize)

	// 验证保留的是最新的项
	_, ok := cache.Get("key4")
	assert.True(t, ok, "newest item should exist")
	_, ok = cache.Get("key3")
	assert.True(t, ok, "second newest item should exist")
}

func TestSettingCache_Expiration(t *testing.T) {
	cache := &settingCache{
		cache:      make(map[string]string),
		lastAccess: make(map[string]time.Time),
		maxSize:    10,
		expiration: 50 * time.Millisecond,
	}

	// 设置项
	cache.Set("test", "value")

	// 立即获取
	v1, ok := cache.Get("test")
	assert.True(t, ok)
	assert.Equal(t, "value", v1)

	// 等待接近过期时间
	time.Sleep(40 * time.Millisecond)

	// 再次访问应该刷新过期时间
	v2, ok := cache.Get("test")
	assert.True(t, ok)
	assert.Equal(t, "value", v2)

	// 等待确保过期
	time.Sleep(60 * time.Millisecond)

	// 应该已经过期
	_, ok = cache.Get("test")
	assert.False(t, ok, "item should have expired")
}

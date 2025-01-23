package conf

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/spf13/viper"
)

// 添加自定义错误类型
var (
	ErrKeyNotFound      = errors.New("setting key not found")
	ErrTypeConversion   = errors.New("type conversion failed")
	ErrStorageOperation = errors.New("storage operation failed")
)

type SettingStorage interface {
	Get(key string) (string, error)
	Set(key, value string) error
	Delete(key string) error
}

// 添加缓存管理器结构体
type settingCache struct {
	cache      map[string]string
	mutex      sync.RWMutex
	lastAccess map[string]time.Time
	maxSize    int
	expiration time.Duration
}

// 为缓存管理器添加方法
func (sc *settingCache) Get(key string) (string, bool) {
	// 首先尝试读取
	sc.mutex.RLock()
	value, ok := sc.cache[key]
	accessTime := sc.lastAccess[key]
	sc.mutex.RUnlock()

	if !ok {
		return "", false
	}

	// 检查是否过期
	if time.Since(accessTime) > sc.expiration {
		sc.mutex.Lock()
		// 双重检查，避免并发问题
		if currentAccessTime, exists := sc.lastAccess[key]; exists {
			if time.Since(currentAccessTime) > sc.expiration {
				delete(sc.cache, key)
				delete(sc.lastAccess, key)
			}
		}
		sc.mutex.Unlock()
		return "", false
	}

	// 更新访问时间
	sc.mutex.Lock()
	sc.lastAccess[key] = time.Now()
	sc.mutex.Unlock()

	return value, true
}

func (sc *settingCache) Set(key, value string) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	// 先清理过期项
	now := time.Now()
	for k, accessTime := range sc.lastAccess {
		if now.Sub(accessTime) > sc.expiration {
			delete(sc.cache, k)
			delete(sc.lastAccess, k)
		}
	}

	// 如果仍然需要清理空间
	if len(sc.cache) >= sc.maxSize {
		// 找到并删除最旧的项
		var oldestKey string
		var oldestTime time.Time
		first := true

		for k, t := range sc.lastAccess {
			if first || t.Before(oldestTime) {
				oldestKey = k
				oldestTime = t
				first = false
			}
		}

		if oldestKey != "" {
			delete(sc.cache, oldestKey)
			delete(sc.lastAccess, oldestKey)
		}
	}

	// 设置新值
	sc.cache[key] = value
	sc.lastAccess[key] = now
}

func (sc *settingCache) Delete(key string) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	delete(sc.cache, key)
}

type SettingManager struct {
	storage SettingStorage
	cache   *settingCache
}

var (
	_settingsManager *SettingManager
	_initOnce        sync.Once
)

// NewSettingManager 创建设置管理器
func NewSettingManager(storage SettingStorage) *SettingManager {
	_initOnce.Do(func() {
		_settingsManager = &SettingManager{
			storage: storage,
			cache: &settingCache{
				cache:      make(map[string]string),
				lastAccess: make(map[string]time.Time),
				maxSize:    1000,
				expiration: 1 * time.Hour,
			},
		}
	})
	return _settingsManager
}

// SetStorage 设置设置存储器
func (sm *SettingManager) SetStorage(storage SettingStorage) {
	sm.storage = storage
}

// Set 设置设置
func (sm *SettingManager) Set(key string, value any) (err error) {
	var strValue string
	switch v := any(value).(type) {
	case string:
		strValue = v
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		strValue = fmt.Sprintf("%d", v)
	case float32:
		strValue = strconv.FormatFloat(float64(v), 'g', -1, 32)
	case float64:
		strValue = strconv.FormatFloat(v, 'g', -1, 64)
	case bool:
		strValue = fmt.Sprintf("%t", v)
	default:
		bytes, err := json.Marshal(value)
		if err != nil {
			return err
		}
		strValue = string(bytes)
	}
	sm.cache.Set(key, strValue)
	err = sm.storage.Set(key, strValue)
	if err != nil {
		return err
	}
	return nil
}

// Set stores a setting with the given key and value
func Set[T any](key string, value T) error {
	return _settingsManager.Set(key, value)
}

func (sm *SettingManager) Get(key string) (any, error) {
	// 先检查缓存
	if value, ok := sm.cache.Get(key); ok {
		return value, nil
	}

	// 从存储层获取
	value, err := sm.storage.Get(key)
	if err != nil {
		if err == ErrKeyNotFound {
			// 尝试从 viper 获取
			if viper.IsSet(key) {
				result := viper.Get(key)
				// 找到值后保存到存储层
				if err := sm.Set(key, result); err != nil {
					return nil, err
				}
				return result, nil
			}
		}
		return nil, err
	}

	// 更新缓存
	sm.cache.Set(key, value)
	return value, nil
}

func MustGet[T any](key string) *T {
	res, err := Get[T](key)
	if err != nil {
		panic(err)
	}
	if res == nil {
		panic(fmt.Errorf("not found config for %s", key))
	}
	return res
}

// Get retrieves a setting by key
func Get[T any](key string) (*T, error) {
	if _settingsManager == nil {
		return nil, fmt.Errorf("settings manager not initialized")
	}

	value, err := _settingsManager.Get(key)
	if err != nil {
		return nil, err
	}

	// 如果值是字符串，尝试解析
	if strValue, ok := value.(string); ok {
		return parseValue[T](strValue)
	}

	// 如果类型已经匹配，直接返回
	if typed, ok := value.(T); ok {
		return &typed, nil
	}

	// 尝试通过 JSON 转换
	jsonData, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	var result T
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	return &result, nil
}

func (sm *SettingManager) Delete(key string) error {
	sm.cache.Delete(key)
	return sm.storage.Delete(key)
}

// Delete removes a setting by key
func Delete(key string) error {
	return _settingsManager.Delete(key)
}

// TypeParser 定义类型转换函数的接口
type TypeParser[T any] func(string) (T, error)

// 定义全局类型转换器映射
var typeParserMap = make(map[string]any)

func init() {
	// 注册基础类型转换器
	registerParser[string](func(v string) (string, error) { return v, nil })
	registerParser[int](func(v string) (int, error) { return strconv.Atoi(v) })
	registerParser[int64](func(v string) (int64, error) { return strconv.ParseInt(v, 10, 64) })
	registerParser[float64](func(v string) (float64, error) { return strconv.ParseFloat(v, 64) })
	registerParser[bool](func(v string) (bool, error) { return strconv.ParseBool(v) })
	registerParser[time.Time](func(v string) (time.Time, error) { return time.Parse(time.RFC3339, v) })
	registerParser[time.Duration](func(v string) (time.Duration, error) { return time.ParseDuration(v) })
}

// registerParser 注册类型转换器
func registerParser[T any](parser TypeParser[T]) {
	t := new(T)
	typeParserMap[fmt.Sprintf("%T", *t)] = parser
}

// parseValue 优化后的实现
func parseValue[T any](value string) (*T, error) {
	var result T
	resultType := fmt.Sprintf("%T", result)

	// 尝试使用注册的解析器
	if parser, ok := typeParserMap[resultType]; ok {
		if typedParser, ok := parser.(TypeParser[T]); ok {
			parsed, err := typedParser(value)
			if err != nil {
				return nil, fmt.Errorf("parse %s error: %w", resultType, err)
			}
			return &parsed, nil
		}
	}

	// 对于复杂类型，使用 JSON 解析
	if err := json.Unmarshal([]byte(value), &result); err != nil {
		return nil, fmt.Errorf("unmarshal complex type error: %w", err)
	}
	return &result, nil
}

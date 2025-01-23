# 配置管理系统 (Configuration Management System)

一个高性能、类型安全的 Go 配置管理系统。

[![Go Report Card](https://goreportcard.com/badge/github.com/hope183/conf)](https://goreportcard.com/report/github.com/hope183/conf)
[![GoDoc](https://godoc.org/github.com/hope183/conf?status.svg)](https://godoc.org/github.com/hope183/conf)

## 特性

- 🔒 线程安全
- 🎯 类型安全的配置读写
- 💾 可扩展的存储后端
- 📦 内置缓存机制
- ⚡ 高性能
- 🧪 完整的测试覆盖

## 安装

```bash
go get github.com/hope183/conf
```

## 快速开始

### 基础使用

```go
package main

import (
    "github.com/hope183/conf"
)

func main() {
    // 创建存储实例
    storage := conf.NewSQLiteStorage("config.db")
    manager := conf.NewSettingManager(storage)

    // 设置配置
    err := conf.Set("app.name", "我的应用")
    if err != nil {
        panic(err)
    }

    // 获取配置
    appName, err := conf.Get[string]("app.name")
    if err != nil {
        panic(err)
    }
    fmt.Printf("应用名称: %s\n", *appName)
}
```

### 结构体配置

```go
type DatabaseConfig struct {
    Host     string `json:"host"`
    Port     int    `json:"port"`
    Username string `json:"username"`
}

// 设置结构体配置
dbConfig := DatabaseConfig{
    Host:     "localhost",
    Port:     5432,
    Username: "admin",
}
err := conf.Set("database", dbConfig)

// 获取结构体配置
config, err := conf.Get[DatabaseConfig]("database")
```

## 详细文档

### 核心 API

#### 设置配置

```go
// 设置字符串
err := Set("app.name", "我的应用")

// 设置数值
err := Set("max.connections", 100)

// 设置布尔值
err := Set("debug.mode", true)
```

#### 获取配置

```go
// 获取字符串
strVal, err := Get[string]("app.name")

// 获取数值
intVal, err := Get[int]("max.connections")

// 获取布尔值
boolVal, err := Get[bool]("debug.mode")
```

#### 删除配置

```go
err := Delete("app.name")
```

### 错误处理

```go
value, err := Get[string]("app.name")
if err != nil {
    switch {
    case errors.Is(err, ErrKeyNotFound):
        // 处理键不存在
    case errors.Is(err, ErrTypeConversion):
        // 处理类型转换错误
    default:
        // 处理其他错误
    }
}
```

### 自定义存储实现

实现 `SettingStorage` 接口来创建自定义存储：

```go
type SettingStorage interface {
    Get(key string) (string, error)
    Set(key, value string) error
    Delete(key string) error
}
```

## 最佳实践

### 配置键命名规范

```go
app.name      // 推荐
app.db.host   // 推荐
APP_NAME      // 不推荐
```

### 批量操作

```go
configs := map[string]interface{}{
    "app.name": "MyApp",
    "app.port": 8080,
}
for k, v := range configs {
    if err := Set(k, v); err != nil {
        // 处理错误
    }
}
```

## 注意事项

1. **缓存一致性**

   - 多实例部署时注意缓存同步
   - 合理设置缓存过期时间

2. **性能优化**

   - 避免频繁读写
   - 合理使用批量操作
   - 适当配置缓存参数

3. **安全建议**
   - 敏感配置加密存储
   - 实现访问控制
   - 定期备份配置

## 常见问题

1. Q: 配置更新后其他实例感知不到？
   A: 考虑实现配置变更通知机制，或设置合理的缓存过期时间。

2. Q: 类型转换失败？
   A: 确保存储的值格式正确，使用正确的类型获取配置。

3. Q: 并发访问性能问题？
   A: 适当调整缓存参数，必要时可以实现分级缓存。

## 贡献指南

欢迎提交 Pull Request 或 Issue！

1. Fork 本仓库
2. 创建特性分支：`git checkout -b my-new-feature`
3. 提交改动：`git commit -am 'Add some feature'`
4. 推送分支：`git push origin my-new-feature`
5. 提交 Pull Request

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

## 版本历史

- v1.0.0
  - 初始发布
  - 基础功能实现
  - 完整的测试覆盖

## 作者

- 作者名字 - [GitHub](https://github.com/hope183)

## 致谢

感谢所有贡献者的付出！

```

```

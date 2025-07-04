# IGo [![Go Report Card](https://goreportcard.com/badge/github.com/aichy126/igo)](https://goreportcard.com/report/github.com/aichy126/igo) [![Language](https://img.shields.io/badge/Language-Go-blue.svg)](https://golang.org/) ![GitHub](https://img.shields.io/github/license/aichy126/igo)

一个现代化的 Go Web 项目脚手架，提供完整的应用生命周期管理、分布式追踪、优雅关闭等功能，让您专注于业务逻辑开发。

## ✨ 特性

- 🚀 **开箱即用** - 预配置常用组件，快速启动项目
- 🔄 **优雅关闭** - 自动处理信号，确保请求完成
- 🔍 **分布式追踪** - 内置 traceId 支持，全链路追踪
- 📊 **健康检查** - 标准化的健康检查和监控接口
- 🛡️ **生产就绪** - 错误处理、中间件、配置管理等
- 📝 **结构化日志** - 基于 zap 的高性能日志系统

## 🏗️ 项目结构

```
igo/
├── igo.go                    # 主入口文件
├── lifecycle/               # 生命周期管理
├── config/                  # 配置管理
├── context/                 # 上下文管理
├── db/                      # 数据库操作
├── cache/                   # 缓存操作
├── log/                     # 日志系统
├── web/                     # Web 框架
├── trace/                   # 分布式追踪
├── httpclient/              # HTTP 客户端
├── util/                    # 工具函数
├── errors.go               # 错误处理
├── example/                 # 示例项目
└── docs/                    # 文档
```

## 🏗️ 包含组件

| 组件 | 描述 | 版本 |
|------|------|------|
| [Gin](https://github.com/gin-gonic/gin) | Web 框架 | 最新 |
| [XORM](https://xorm.io/) | ORM 框架 | 最新 |
| [Zap](https://github.com/uber-go/zap) | 日志库 | 最新 |
| [Viper](https://github.com/spf13/viper) | 配置管理 | 最新 |
| [Redis](https://github.com/go-redis/redis) | 缓存客户端 | v8 |
| [PPROF](https://golang.org/pkg/net/http/pprof/) | 性能分析 | 内置 |

## 🚀 快速开始

### 安装

```bash
go get github.com/aichy126/igo
```

### 基本使用

```go
package main

import (
	"github.com/aichy126/igo"
	"github.com/aichy126/igo/context"
	"github.com/aichy126/igo/log"
	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化应用
	igo.App = igo.NewApp("config.toml")

	// 添加生命周期钩子
	igo.App.AddStartupHook(func() error {
		log.Info("应用启动")
		return nil
	})

	// 设置路由
	igo.App.Web.Router.GET("/ping", Ping)

	// 使用生命周期管理器运行
	if err := igo.App.RunWithLifecycle(); err != nil {
		log.Error("应用运行失败", log.Any("error", err))
	}
}

func Ping(c *gin.Context) {
	ctx := context.Ginform(c)
	ctx.LogInfo("收到 ping 请求") // 自动带 traceId

	c.JSON(200, gin.H{
		"message": "pong",
		"traceId": c.GetString("traceId"),
	})
}
```

### 配置文件

```toml
[local]
address = ":8001"
debug = true
shutdown_timeout = 30

[local.logger]
dir = "./logs"
name = "log.log"
access = true
level = "INFO"
max_size = 1
max_backups = 5
max_age = 7

[mysql.igo]
max_idle = 10
max_open = 20
is_debug = true
data_source = "root:root@tcp(127.0.0.1:3306)/igo?charset=utf8mb4&parseTime=True&loc=Local"

[redis.igorediskey]
address = "127.0.0.1:6379"
password = ""
db = 0
poolsize = 50
```

## 🔧 核心功能

### 1. 应用生命周期管理

```go
// 启动钩子
igo.App.AddStartupHook(func() error {
	log.Info("初始化外部资源")
	return nil
})

// 关闭钩子
igo.App.AddShutdownHook(func() error {
	log.Info("清理资源")
	return nil
})
```

### 2. 分布式追踪

```go
func handler(c *gin.Context) {
	ctx := context.Ginform(c)

	// 自动带 trace 信息的日志
	ctx.LogInfo("业务处理开始", log.String("user_id", "12345"))

	// 创建业务 span
	ctx = ctx.StartBusinessSpan("用户登录")
	defer ctx.EndBusinessSpan(nil)

	// 错误日志也自动带 trace 信息
	if err != nil {
		ctx.LogError("处理失败", log.String("error", err.Error()))
	}
}
```

### 3. 数据库操作

```go
// 获取数据库实例
db := igo.App.DB.NewDBTable("igo", "users")

// 查询数据
var users []User
err := db.Where("status = ?", 1).Find(&users)

// 事务操作
session := db.NewSession()
defer session.Close()

err = session.Begin()
// ... 业务逻辑
err = session.Commit()
```

### 4. 缓存操作

```go
// 获取 Redis 客户端
redis, err := igo.App.Cache.Get("igorediskey")
if err != nil {
	return err
}

// 设置缓存
err = redis.Set(ctx, "key", "value", time.Hour).Err()

// 获取缓存
value, err := redis.Get(ctx, "key").Result()
```

### 5. 配置管理

```go
// 读取配置
debug := util.ConfGetbool("local.debug")
address := util.ConfGetString("local.address")

// 直接使用 viper
port := igo.App.Conf.GetInt("local.port")
```

## 📊 监控接口

### 健康检查
```bash
curl http://localhost:8001/health
# {"status":"ok","time":"2024-01-01T12:00:00Z"}
```

### 就绪检查
```bash
curl http://localhost:8001/ready
# {"status":"ready","time":"2024-01-01T12:00:00Z"}
```

### 性能分析
```bash
# 访问 pprof 接口（debug 模式）
curl http://localhost:8001/debug/pprof/
```

## 📚 详细文档

详细的使用说明和最佳实践请查看 [docs/](docs/) 目录：

- **[📖 文档索引](docs/README.md)** - 完整的文档导航
- **[🚀 生命周期管理](docs/lifecycle.md)** - 应用启动、关闭、钩子等
- **[🔍 分布式追踪](docs/tracing.md)** - traceId、span、日志集成等
- **[⚙️ 配置管理](docs/configuration.md)** - 配置文件、环境变量等
- **[🗄️ 数据库操作](docs/database.md)** - ORM 使用、事务等
- **[💾 缓存操作](docs/cache.md)** - Redis 使用、连接池等
- **[🛡️ 错误处理](docs/error-handling.md)** - 错误类型、错误码等

## 🛠️ 开发

### 运行示例

```bash
cd example
go run main.go
```

### 测试接口

```bash
# 基本接口
curl http://localhost:8001/ping

# 业务 span 示例
curl http://localhost:8001/business-span

# 追踪示例
curl http://localhost:8001/trace-examples
```

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🔗 相关链接

- [Go 官方文档](https://golang.org/doc/)
- [Gin 框架文档](https://gin-gonic.com/docs/)
- [XORM 文档](https://xorm.io/docs/)
- [Zap 日志库](https://github.com/uber-go/zap)

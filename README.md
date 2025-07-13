# IGo [![Go Report Card](https://goreportcard.com/badge/github.com/aichy126/igo)](https://goreportcard.com/report/github.com/aichy126/igo) [![Language](https://img.shields.io/badge/Language-Go-blue.svg)](https://golang.org/) ![GitHub](https://img.shields.io/github/license/aichy126/igo)

一个轻量级的 Go Web 项目脚手架，专注于基础组件初始化和协调，让您快速搭建项目基础设施。

## ✨ 设计原则

- 🎯 **简单易用** - 方法名通俗易懂，链式调用，开箱即用
- 🔒 **职责边界清晰** - 只负责基础组件初始化，不干预业务逻辑
- 🪶 **轻量灵活** - 最小化依赖，配置驱动，支持扩展
- 🔇 **容错设计** - 可选组件（redis/xorm）初始化失败时静默处理

## 🏗️ 核心功能

| 功能 | 描述 | 状态 |
|------|------|------|
| **应用生命周期** | 优雅启动/关闭、钩子管理、信号处理 | ✅ 核心功能 |
| **配置热重载** | 文件实时监听、Consul轮询、手动重载 | ✅ 核心功能 |
| **分布式追踪** | traceId自动传递、日志关联、上下文传递 | ✅ 核心功能 |
| **HTTP客户端** | 链式配置、自动重试、header管理、trace传递 | ✅ 核心功能 |
| **Web框架** | 基于Gin、中间件、pprof集成 | ✅ 核心功能 |
| **日志系统** | 结构化日志、trace集成、文件轮转 | ✅ 核心功能 |
| **数据库** | XORM集成、多数据源、连接池 | 🔸 可选组件 |
| **缓存** | Redis集成、连接池管理 | 🔸 可选组件 |

## 🚀 快速开始

### 安装

```bash
go get github.com/aichy126/igo
```

### 基本使用

```go
package main

import (
	"log"

	"github.com/aichy126/igo"
	"github.com/aichy126/igo/ictx"
	ilog "github.com/aichy126/igo/ilog"
	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化应用
	app, err := igo.NewApp("")
	if err != nil {
		log.Fatal("应用初始化失败:", err)
	}
	igo.App = app

	// 配置热重载（文件自动启用，Consul需手动设置）
	if app.Conf.GetString("config.address") != "" {
		app.SetConfigHotReloadInterval(60) // Consul配置60秒轮询
	}

	// 添加配置变更回调
	app.AddConfigChangeCallback(func() {
		ilog.Info("配置已更新", 
			ilog.Bool("debug", app.Conf.GetBool("local.debug")))
	})

	// 添加生命周期钩子
	app.AddStartupHook(func() error {
		ilog.Info("应用启动完成")
		return nil
	})

	// 设置路由
	app.Web.Router.GET("/ping", Ping)
	app.Web.Router.POST("/reload-config", ReloadConfig)

	// 运行应用（自动处理优雅关闭）
	if err := app.Run(); err != nil {
		ilog.Error("应用运行失败", ilog.Any("error", err))
	}
}

func Ping(c *gin.Context) {
	ctx := ictx.Ginform(c)
	ctx.LogInfo("收到ping请求") // 自动带traceId

	c.JSON(200, gin.H{
		"message": "pong",
		"traceId": c.GetString("traceId"),
	})
}

func ReloadConfig(c *gin.Context) {
	ctx := ictx.Ginform(c)
	if err := igo.App.ReloadConfig(); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	ctx.LogInfo("配置重载成功")
	c.JSON(200, gin.H{"message": "配置已重载"})
}
```

### 配置文件

```toml
[local]
address = ":8001"
debug = true  # 开启时自动启用pprof性能分析

[local.logger]
dir = "./logs"
name = "log.log"
level = "INFO"

# 可选：数据库配置
[mysql.igo]
max_idle = 10
max_open = 20
data_source = "root:root@tcp(127.0.0.1:3306)/igo?charset=utf8mb4"

# 可选：Redis配置
[redis.igorediskey]
address = "127.0.0.1:6379"
db = 0

# 可选：Consul配置
[config]
address = "127.0.0.1:8500"
key = "igo/config"
```

## 🔧 核心功能详解

### 1. 应用生命周期管理

```go
// 自动优雅关闭
if err := app.Run(); err != nil {
	log.Fatal(err)
}

// 自定义钩子
app.AddStartupHook(func() error {
	// 启动时执行
	return nil
})

app.AddShutdownHook(func() error {
	// 关闭时执行
	return nil
})
```

### 2. 配置热重载

```go
// 文件配置：自动启用热重载
// Consul配置：手动设置轮询间隔
app.SetConfigHotReloadInterval(60) // 60秒轮询一次

// 配置变更回调
app.AddConfigChangeCallback(func() {
	// 配置更新时执行
})

// 手动重载
app.ReloadConfig()
```

### 3. HTTP客户端

```go
// 链式配置
client := httpclient.New().
	SetDefaultTimeout(time.Second * 30).
	SetDefaultRetries(3).
	SetHeader("Authorization", "Bearer token").
	SetUserAgent("MyApp/1.0").
	SetBaseURL("https://api.example.com")

// 自动重试请求
resp, err := client.Get(ctx, "/users/123")

// 获取字符串响应
content, err := client.GetBodyString(ctx, "GET", "/page.html", nil)
```

### 4. 分布式追踪

```go
import (
	"github.com/aichy126/igo/ictx"
	ilog "github.com/aichy126/igo/ilog"
	"github.com/gin-gonic/gin"
)

func Handler(c *gin.Context) {
	ctx := ictx.Ginform(c)
	
	// 自动带traceId的日志
	ctx.LogInfo("处理开始")
	
	// 设置业务数据
	ctx.Set("user_id", "12345")
	ctx.Set("operation", "user_login")
	
	// 记录业务日志（自动包含traceId）
	ctx.LogInfo("用户登录处理", 
		ilog.String("user_id", ctx.GetString("user_id")),
		ilog.String("operation", ctx.GetString("operation")))
	
	// HTTP调用自动传递traceId
	client.Get(ctx, "/external-api")
}
```

## 📊 测试接口

启动示例应用后：

```bash
# 基本功能
curl http://localhost:8001/ping
curl http://localhost:8001/health
curl http://localhost:8001/ready

# 配置热重载
curl -X POST http://localhost:8001/reload-config

# 性能分析（debug模式）
curl http://localhost:8001/debug/pprof/
```

## 🛠️ 开发命令

```bash
# 运行示例
cd example && go run main.go

# 构建项目
go build ./...

# 运行测试
go test ./...

# 格式化代码
go fmt ./...
```

## 🎯 明确不包含的功能

为保持脚手架的简洁性，以下功能不在范围内：

- **开发工具**: 代码生成、数据库迁移、Docker配置
- **监控系统**: 指标收集、性能监控、链路跟踪服务端
- **高级功能**: 分布式锁、消息队列、服务发现

这些功能由业务项目根据实际需求自行实现。

## 📚 文档

- **[错误处理规范](docs/error-handling.md)** - 脚手架错误处理原则
- **[生命周期管理](docs/lifecycle.md)** - 钩子和信号处理
- **[配置管理](docs/configuration.md)** - 配置文件和热重载
- **[数据库使用](docs/database.md)** - XORM使用指南
- **[缓存使用](docs/cache.md)** - Redis使用指南

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。
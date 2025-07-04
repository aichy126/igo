# 生命周期管理

> **注意**: `lifecycle.go` 文件已移动到 `lifecycle/` 目录下，通过接口设计避免循环导入问题。

## 概述

igo脚手架现在提供了完整的应用生命周期管理功能，包括优雅启动、优雅关闭、信号处理、分布式追踪（traceId）等。

## 主要功能

### 1. 优雅关闭
- 自动处理SIGINT和SIGTERM信号
- 按顺序关闭各个组件（Web服务器、数据库、缓存）
- 支持自定义关闭超时时间
- 确保正在处理的请求能够完成

### 2. 生命周期钩子
- 启动钩子：应用启动时执行
- 关闭钩子：应用关闭时执行
- 支持多个钩子的顺序执行

### 3. 健康检查
- `/health` - 健康检查接口
- `/ready` - 就绪检查接口

### 4. 分布式追踪
- HTTP请求自动生成和透传 traceId
- 支持自定义span、事件、属性
- 详细说明请参考 [分布式追踪文档](tracing.md)

## 使用方法

### 基本使用

```go
package main

import (
    "github.com/aichy126/igo"
    "github.com/aichy126/igo/log"
)

func main() {
    // 初始化应用
    igo.App = igo.NewApp("config.toml")

    // 设置路由
    Router(igo.App.Web.Router)

    // 使用生命周期管理器运行
    if err := igo.App.RunWithLifecycle(); err != nil {
        log.Error("应用运行失败", log.Any("error", err))
    }
}
```

### 添加生命周期钩子

```go
func main() {
    igo.App = igo.NewApp("config.toml")

    // 添加启动钩子
    igo.App.AddStartupHook(func() error {
        log.Info("应用启动钩子执行")
        // 执行启动时的初始化工作
        return nil
    })

    // 添加关闭钩子
    igo.App.AddShutdownHook(func() error {
        log.Info("应用关闭钩子执行")
        // 执行关闭时的清理工作
        return nil
    })

    // 运行应用
    igo.App.RunWithLifecycle()
}
```

### 手动优雅关闭

```go
// 手动触发优雅关闭
if err := igo.App.GracefulShutdown(30 * time.Second); err != nil {
    log.Error("优雅关闭失败", log.Any("error", err))
}
```

## 关闭顺序

应用关闭时，会按以下顺序执行：

1. 执行关闭钩子（反向顺序）
2. 关闭Web服务器（停止接受新请求，等待现有请求完成）
3. 关闭数据库连接
4. 关闭缓存连接

## 配置

可以在配置文件中设置关闭超时时间：

```toml
[local]
address = ":8091"
debug = true
shutdown_timeout = 30  # 关闭超时时间（秒）

[local.logger]
dir = "./logs"
name = "log.log"
access = true
level = "INFO"
max_size = 1
max_backups = 5
max_age = 7
```

## 监控接口

### 健康检查
```bash
curl http://localhost:8091/health
```

响应：
```json
{
    "status": "ok",
    "time": "2024-01-01T12:00:00Z"
}
```

### 就绪检查
```bash
curl http://localhost:8091/ready
```

响应：
```json
{
    "status": "ready",
    "time": "2024-01-01T12:00:00Z"
}
```

## 最佳实践

1. **启动钩子**：用于初始化外部资源、预热缓存、加载配置等
2. **关闭钩子**：用于保存状态、清理临时文件、通知其他服务等
3. **超时设置**：根据应用复杂度设置合适的关闭超时时间
4. **错误处理**：在钩子中妥善处理错误，避免影响应用关闭流程
5. **日志记录**：在关键步骤添加日志，便于问题排查
6. **分布式追踪**：所有日志、RPC、DB操作建议都带上 traceId，便于全链路追踪

## 注意事项

1. 关闭钩子中的操作应该是幂等的
2. 避免在钩子中执行长时间阻塞的操作
3. 确保所有外部连接都能正确关闭
4. 在生产环境中建议设置合理的关闭超时时间
5. 分布式追踪建议全链路透传，便于排查分布式问题

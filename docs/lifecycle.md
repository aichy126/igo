# 生命周期管理

## 概述

IGo 脚手架提供简化的应用生命周期管理功能，专注于钩子管理和信号处理，不干预具体的组件管理。

## 设计原则

- **简化职责**: 只处理钩子执行和信号监听，不直接管理具体组件
- **灵活扩展**: 通过钩子机制支持自定义初始化和清理逻辑
- **优雅关闭**: 监听系统信号，确保应用能够优雅退出

## 主要功能

### 1. 生命周期钩子

支持启动和关闭钩子的注册和执行：

```go
// 启动钩子：应用启动时执行
igo.App.AddStartupHook(func() error {
    log.Info("初始化外部资源")
    return nil
})

// 关闭钩子：应用关闭时执行
igo.App.AddShutdownHook(func() error {
    log.Info("清理资源")
    return nil
})
```

### 2. 信号处理

自动监听系统信号：
- `SIGINT` (Ctrl+C)
- `SIGTERM` (终止信号)

### 3. 简化的应用接口

```go
// 运行应用（等待信号）
err := igo.App.Run()

// 手动关闭应用
err := igo.App.Shutdown()
```

## 使用方法

### 基本使用

```go
package main

import (
    "log"
    "github.com/aichy126/igo"
    ilog "github.com/aichy126/igo/ilog"
)

func main() {
    // 初始化应用
    app, err := igo.NewApp("")
    if err != nil {
        log.Fatal("应用初始化失败:", err)
    }
    igo.App = app

    // 添加启动钩子
    igo.App.AddStartupHook(func() error {
        ilog.Info("应用启动完成")
        return nil
    })

    // 添加关闭钩子
    igo.App.AddShutdownHook(func() error {
        ilog.Info("应用正在关闭")
        return nil
    })

    // 设置路由
    setupRoutes(igo.App.Web.Router)

    // 运行应用（自动启动Web服务器并处理优雅关闭）
    if err := igo.App.Run(); err != nil {
        ilog.Error("应用运行失败", ilog.Any("error", err))
    }
}
```

### 钩子执行顺序

1. **启动时**: 按添加顺序执行启动钩子
2. **关闭时**: 按添加顺序的**逆序**执行关闭钩子

```go
// 添加顺序：Hook1 -> Hook2 -> Hook3
igo.App.AddShutdownHook(hook1)
igo.App.AddShutdownHook(hook2) 
igo.App.AddShutdownHook(hook3)

// 执行顺序：Hook3 -> Hook2 -> Hook1
```

### 钩子最佳实践

#### 启动钩子示例

```go
// 数据库连接验证
igo.App.AddStartupHook(func() error {
    if igo.App.DB != nil {
        // 验证数据库连接
        return validateDatabaseConnection()
    }
    return nil
})

// 外部服务健康检查
igo.App.AddStartupHook(func() error {
    return checkExternalServices()
})
```

#### 关闭钩子示例

```go
// 完成正在处理的任务
igo.App.AddShutdownHook(func() error {
    return finishPendingTasks()
})

// 清理临时文件
igo.App.AddShutdownHook(func() error {
    return cleanupTempFiles()
})

// 关闭外部连接
igo.App.AddShutdownHook(func() error {
    return closeExternalConnections()
})
```

## 错误处理

### 启动钩子错误

- 任何启动钩子返回错误会**终止应用启动**
- 建议在钩子中进行适当的错误处理和重试

### 关闭钩子错误

- 关闭钩子的错误会被**记录但不会阻止关闭流程**
- 确保关闭逻辑尽可能简单可靠

## 注意事项

1. **职责边界**: 生命周期管理器只负责钩子执行，不直接管理Web服务器、数据库等组件
2. **组件管理**: Web服务器需要在业务代码中手动启动（通常在goroutine中）
3. **静默失败**: 可选组件（db/cache）的初始化失败不会影响应用启动
4. **钩子简洁**: 保持钩子逻辑简单，避免长时间阻塞操作

## 与原版本的差异

### 简化前（v1.x）
- 复杂的组件接口和应用接口
- 多套关闭机制
- 直接管理Web服务器、数据库等组件

### 简化后（v2.0）
- 简化为4个核心方法：`Run()`, `Shutdown()`, `AddStartupHook()`, `AddShutdownHook()`
- 统一的钩子机制
- 只负责信号处理，不直接管理具体组件

这种设计更好地体现了脚手架的定位：专注于基础设施协调，不干预业务逻辑。
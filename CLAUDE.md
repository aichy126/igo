# CLAUDE.md

这个文件为Claude Code提供IGo项目的开发指导。

## 沟通语言
**重要**: 请始终使用中文与用户交流。所有回复、解释和文档都应该用中文。

## 项目简介

IGo是一个轻量级Go Web框架脚手架，专注于基础组件初始化和协调。

核心设计原则：
- 简单易用，链式调用
- 职责边界清晰，不干预业务逻辑  
- 轻量灵活，最小化依赖
- 容错设计，可选组件失败时静默处理

## 开发命令

```bash
# 运行示例应用
cd example && go run main.go

# 构建项目
go build ./...

# 运行所有测试
go test ./...

# 运行特定包的测试
go test -v -cover ./ictx
go test ./httpclient
go test ./config

# 格式化代码
go fmt ./...

# 下载依赖
go mod tidy
```

## 测试接口 (启动示例应用后)

```bash
# 基本功能测试
curl http://localhost:8001/ping
curl http://localhost:8001/business-span
curl http://localhost:8001/trace-demo
curl http://localhost:8001/health
curl http://localhost:8001/ready

# 配置热重载测试
curl -X POST http://localhost:8001/reload-config

# 性能分析（仅debug模式）
curl http://localhost:8001/debug/pprof/
```

## 核心架构

### 应用编排
`igo.go`包含主要的`Application`结构体，协调所有组件：
- Web服务器
- 数据库连接（可选）
- 缓存连接（可选）
- 配置管理
- 生命周期管理

提供自动优雅关闭：`RunWithGracefulShutdown()`直接处理操作系统信号。

### 带追踪的请求流
所有请求通过中间件自动注入traceId。`ictx.Ginform(c)`函数创建增强的上下文，支持：
- 通过`ctx.LogInfo()`进行追踪感知的日志记录
- 通过`ctx.StartBusinessSpan("name")`创建业务span
- 自动数据传递和管理

### 组件初始化
组件在`NewApp()`期间按顺序初始化：
1. 配置加载和验证
2. Web服务器设置和中间件
3. 日志配置  
4. 数据库连接（可选，失败时静默处理）
5. 缓存/Redis设置（可选，失败时静默处理）
6. 生命周期管理器

### 配置管理
- 支持TOML文件、Consul、环境变量
- 热重载：文件自动监听，Consul用户配置轮询间隔
- 配置变更回调机制

### HTTP客户端
链式配置的HTTP客户端，支持：
- 超时、重试、header、TLS配置
- 自动trace传递
- 根据初始化配置自动决定是否重试

### 数据库模式
`igo.App.DB.NewDBTable(configKey, tableName)`返回特定数据库连接的XORM实例。

### 生命周期钩子
通过`AddStartupHook()`和`AddShutdownHook()`注册启动和关闭钩子。

## 脚手架错误处理

IGo作为脚手架专注于组件初始化和协调，不干预业务逻辑：

**脚手架职责**:
- 安全地初始化各个组件(web/db/cache/config)
- 关键组件失败返回错误，可选组件失败记录警告继续启动
- 避免不当panic，使用安全的类型断言
- 将错误正确传递给业务层

**已修复的问题**:
- `NewApp()`现在返回错误而不是panic
- 危险的类型断言改为安全版本
- 可选组件(db/cache)失败时记录警告但继续启动

**使用示例**:
```go
// 脚手架层：安全的组件初始化
app, err := igo.NewApp("")
if err != nil {
    log.Error("应用初始化失败", log.Any("error", err))
    os.Exit(1)
}

// 业务层：自行处理业务逻辑错误
func CreateUser(ctx ictx.IContext) {
    // 业务逻辑错误处理由业务层决定
    if err := userService.Create(); err != nil {
        // 业务层自定义错误处理策略
    }
}
```

详细规范参见：`docs/error-handling.md`

## 重要约定

1. **不要添加注释**：除非用户明确要求，否则代码中不添加注释
2. **不要创建文档文件**：除非用户明确要求，否则不主动创建.md文件
3. **优先编辑现有文件**：而不是创建新文件
4. **保持简洁**：专注于脚手架功能，不添加业务相关方法
5. **遵循错误处理规范**：关键组件失败返回错误，可选组件失败静默处理
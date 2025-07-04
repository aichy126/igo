# TraceId 使用指南

## 概述

在 igo 脚手架中，使用 `ctx.LogInfo` 和 `ctx.LogError` 可以让日志自动带上 trace 信息，不依赖 gin.Context。

## 快速开始

```go
func handler(c *gin.Context) {
    // 获取带 trace 信息的 context
    ctx := context.Ginform(c)

    // 直接使用 ctx.LogInfo，自动带 trace 信息
    ctx.LogInfo("业务处理开始", log.String("user_id", "12345"))

    // 错误日志也自动带 trace 信息
    if err != nil {
        ctx.LogError("业务处理失败", log.String("error", err.Error()))
    }
}
```

**就是这么简单！** 使用 `ctx.LogInfo` 和 `ctx.LogError` 会自动带上 trace 信息，无需额外配置。

## 使用方式

### 1. 基本使用

```go
func handler(c *gin.Context) {
    ctx := context.Ginform(c)

    // 信息日志
    ctx.LogInfo("开始处理请求", log.String("endpoint", "/api/users"))

    // 错误日志
    if err != nil {
        ctx.LogError("处理失败", log.String("error", err.Error()))
    }
}
```

### 2. 带业务 span 的使用

```go
func handler(c *gin.Context) {
    // 创建带业务span的context，自动继承traceId
    ctx := context.Ginform(c).StartBusinessSpan("用户登录业务")
    defer ctx.EndBusinessSpan(nil) // 自动结束span

    // 自动带 trace 和 span 信息
    ctx.LogInfo("开始用户登录", log.String("user_id", "12345"))

    // 记录业务事件
    if span := ctx.GetBusinessSpan(); span != nil {
        trace.AddEvent(span, "用户验证", map[string]string{
            "user_id": "12345",
            "status":  "success",
        })
    }

    // 错误日志也自动带 trace 信息
    if err != nil {
        ctx.LogError("用户登录失败", log.String("error", err.Error()))
    }
}
```

## 实际应用场景

### 业务函数中传递 context

```go
func BusinessFunction(ctx context.Context) error {
    // 创建子 span
    childCtx, childSpan := trace.GlobalTracer.StartSpan(ctx, "业务操作")
    defer trace.EndSpan(childSpan, nil)

    // 业务逻辑
    // ...

    // 直接使用 ctx.LogInfo，自动带 trace 信息
    childCtx.LogInfo("业务操作完成",
        log.String("operation", "user_login"),
        log.String("user_id", "12345"),
    )

    return nil
}
```

### 异步操作中保持 traceId

```go
func AsyncOperation() {
    ctx, span := trace.GlobalTracer.StartSpan(context.Background(), "异步操作")
    defer trace.EndSpan(span, nil)

    // 启动 goroutine，传递 context
    go func(ctx context.Context) {
        asyncCtx, asyncSpan := trace.GlobalTracer.StartSpan(ctx, "异步子操作")
        defer trace.EndSpan(asyncSpan, nil)

        // 异步操作中的日志仍然带 traceId
        asyncCtx.LogInfo("异步操作执行中")
    }(ctx)
}
```

### 数据库操作中带 traceId

```go
func DatabaseOperation(ctx context.Context) {
    dbCtx, dbSpan := trace.GlobalTracer.StartSpan(ctx, "DB查询",
        trace.WithAttributes(map[string]string{
            "db.operation": "SELECT",
            "db.table":     "users",
        }),
    )
    defer trace.EndSpan(dbSpan, nil)

    // 数据库操作日志
    dbCtx.LogInfo("执行数据库查询",
        log.String("sql", "SELECT * FROM users WHERE id = ?"),
    )
}
```

## 测试示例

运行示例项目并访问以下接口查看效果：

```bash
# 启动示例项目
go run example/main.go

# 访问基本日志示例接口
curl http://localhost:8091/ping

# 访问业务span示例接口
curl http://localhost:8091/business-span

# 访问 traceId 示例接口
curl http://localhost:8091/trace-examples
```

## 日志输出示例

使用 ctx.LogInfo 的日志输出格式：

```json
{
    "level": "INFO",
    "time": "2024-01-01T12:00:00.000Z",
    "msg": "这是带 trace 信息的日志",
    "trace_id": "550e8400-e29b-41d4-a716-446655440000",
    "span_id": "550e8400-e29b-41d4-a716-446655440001",
    "span_name": "用户登录业务",
    "key": "value"
}
```

## 最佳实践

1. **使用 ctx.LogInfo/LogError**：自动带 trace 信息，代码最简洁
2. **使用 context.Ginform(c).StartBusinessSpan()**：自动继承 traceId，创建业务span
3. **在函数参数中传递 context**：确保 traceId 能够正确传递
4. **为重要操作创建子 span**：便于细粒度追踪
5. **在异步操作中传递 context**：保持 traceId 连续性
6. **记录关键业务事件**：使用 trace.AddEvent 记录重要节点
7. **使用 defer 自动结束 span**：避免忘记结束span

## 注意事项

1. 确保在函数开始时创建 span，结束时调用 EndSpan
2. 在 goroutine 中要传递 context，否则会丢失 traceId
3. 错误处理时要正确设置 span 状态
4. 避免在循环中频繁创建 span，影响性能

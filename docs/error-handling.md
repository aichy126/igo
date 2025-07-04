# 错误处理

## 概述

IGo 提供了标准化的错误处理机制，包括错误类型定义、错误码管理、HTTP 状态码映射等，确保错误响应的一致性和可维护性。

## 错误类型

### AppError

IGo 定义了 `AppError` 类型来表示应用错误：

```go
type AppError struct {
    Code    int         `json:"code"`              // 错误码
    Message string      `json:"message"`           // 错误消息
    Details interface{} `json:"details,omitempty"` // 错误详情
    Err     error      `json:"-"`                  // 原始错误
}
```

### 错误码定义

```go
const (
    // 系统级错误码 (1000-1999)
    ErrCodeSystemError        = 1000 // 系统错误
    ErrCodeConfigError        = 1001 // 配置错误
    ErrCodeDatabaseError      = 1002 // 数据库错误
    ErrCodeCacheError         = 1003 // 缓存错误
    ErrCodeNetworkError       = 1004 // 网络错误
    ErrCodeTimeoutError       = 1005 // 超时错误

    // 业务级错误码 (2000-2999)
    ErrCodeValidationError    = 2000 // 参数验证错误
    ErrCodeNotFoundError      = 2001 // 资源不存在
    ErrCodeUnauthorizedError  = 2002 // 未授权
    ErrCodeForbiddenError     = 2003 // 禁止访问
    ErrCodeConflictError      = 2004 // 资源冲突
    ErrCodeRateLimitError     = 2005 // 限流错误
)
```

## 基本使用

### 创建错误

```go
import "github.com/aichy126/igo"

// 创建简单错误
err := igo.NewError(igo.ErrCodeValidationError, "参数验证失败")

// 创建带详情的错误
err = igo.NewErrorWithDetails(igo.ErrCodeValidationError, "参数验证失败", map[string]interface{}{
    "field": "email",
    "value": "invalid-email",
})

// 包装现有错误
err = igo.WrapError(igo.ErrCodeDatabaseError, "数据库查询失败", originalErr)
```

### 错误检查

```go
// 检查错误类型
if appErr, ok := err.(*igo.AppError); ok {
    log.Error("应用错误",
        log.Int("code", appErr.Code),
        log.String("message", appErr.Message),
    )
}

// 检查错误码
if igo.IsErrorCode(err, igo.ErrCodeNotFoundError) {
    // 处理资源不存在错误
}

// 获取错误码
code := igo.GetErrorCode(err)
```

### HTTP 错误响应

```go
func handler(c *gin.Context) {
    // 业务逻辑
    if err != nil {
        // 自动转换为标准 JSON 响应
        igo.HandleError(c, err)
        return
    }

    c.JSON(200, gin.H{"data": result})
}
```

## 错误处理中间件

### 自动错误处理

IGo 提供了错误处理中间件，自动处理错误并返回标准化的 JSON 响应：

```go
// 在路由中注册中间件
r.Use(web.ErrorHandler())

// 在业务代码中抛出错误
func handler(c *gin.Context) {
    if err != nil {
        // 中间件会自动处理错误
        panic(err)
    }
}
```

### 错误响应格式

```json
{
    "code": 2001,
    "message": "用户不存在",
    "details": {
        "user_id": 12345
    },
    "timestamp": "2024-01-01T12:00:00Z",
    "request_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

## 错误码映射

### HTTP 状态码映射

IGo 自动将错误码映射到相应的 HTTP 状态码：

```go
// 错误码到 HTTP 状态码的映射
var ErrorCodeToHTTPStatus = map[int]int{
    ErrCodeSystemError:       http.StatusInternalServerError,    // 500
    ErrCodeConfigError:       http.StatusInternalServerError,    // 500
    ErrCodeDatabaseError:     http.StatusInternalServerError,    // 500
    ErrCodeCacheError:        http.StatusInternalServerError,    // 500
    ErrCodeNetworkError:      http.StatusBadGateway,             // 502
    ErrCodeTimeoutError:      http.StatusGatewayTimeout,         // 504
    ErrCodeValidationError:   http.StatusBadRequest,             // 400
    ErrCodeNotFoundError:     http.StatusNotFound,               // 404
    ErrCodeUnauthorizedError: http.StatusUnauthorized,           // 401
    ErrCodeForbiddenError:    http.StatusForbidden,              // 403
    ErrCodeConflictError:     http.StatusConflict,               // 409
    ErrCodeRateLimitError:    http.StatusTooManyRequests,        // 429
}
```

## 业务错误处理

### 参数验证错误

```go
func CreateUser(c *gin.Context) {
    var user User
    if err := c.ShouldBindJSON(&user); err != nil {
        // 参数验证失败
        igo.HandleError(c, igo.NewErrorWithDetails(
            igo.ErrCodeValidationError,
            "参数验证失败",
            map[string]interface{}{
                "error": err.Error(),
                "field": "request_body",
            },
        ))
        return
    }

    // 业务逻辑...
}
```

### 资源不存在错误

```go
func GetUser(c *gin.Context) {
    id := c.Param("id")

    user, err := userService.GetByID(id)
    if err != nil {
        if errors.Is(err, userService.ErrUserNotFound) {
            igo.HandleError(c, igo.NewErrorWithDetails(
                igo.ErrCodeNotFoundError,
                "用户不存在",
                map[string]interface{}{
                    "user_id": id,
                },
            ))
            return
        }

        // 其他错误
        igo.HandleError(c, igo.WrapError(igo.ErrCodeDatabaseError, "查询用户失败", err))
        return
    }

    c.JSON(200, gin.H{"data": user})
}
```

### 权限错误

```go
func UpdateUser(c *gin.Context) {
    userID := c.Param("id")
    currentUser := getCurrentUser(c)

    // 检查权限
    if currentUser.ID != userID && !currentUser.IsAdmin {
        igo.HandleError(c, igo.NewError(
            igo.ErrCodeForbiddenError,
            "没有权限修改此用户",
        ))
        return
    }

    // 业务逻辑...
}
```

## 错误日志记录

### 结构化错误日志

```go
func handler(c *gin.Context) {
    ctx := context.Ginform(c)

    if err != nil {
        // 记录错误日志，自动带 trace 信息
        ctx.LogError("业务处理失败",
            log.String("error", err.Error()),
            log.Any("request_id", c.GetString("traceId")),
        )

        igo.HandleError(c, err)
        return
    }
}
```

### 错误分类记录

```go
func logError(ctx context.Context, err error) {
    if appErr, ok := err.(*igo.AppError); ok {
        switch appErr.Code {
        case igo.ErrCodeValidationError:
            // 参数验证错误，记录为警告
            ctx.LogWarn("参数验证失败",
                log.Int("code", appErr.Code),
                log.String("message", appErr.Message),
                log.Any("details", appErr.Details),
            )
        case igo.ErrCodeNotFoundError:
            // 资源不存在，记录为信息
            ctx.LogInfo("资源不存在",
                log.Int("code", appErr.Code),
                log.String("message", appErr.Message),
            )
        default:
            // 其他错误，记录为错误
            ctx.LogError("业务错误",
                log.Int("code", appErr.Code),
                log.String("message", appErr.Message),
                log.Any("details", appErr.Details),
            )
        }
    } else {
        // 系统错误
        ctx.LogError("系统错误",
            log.String("error", err.Error()),
        )
    }
}
```

## 错误恢复

### Panic 恢复

```go
func RecoveryMiddleware() gin.HandlerFunc {
    return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
        if err, ok := recovered.(string); ok {
            // 字符串类型的 panic
            igo.HandleError(c, igo.NewError(igo.ErrCodeSystemError, err))
        } else if err, ok := recovered.(error); ok {
            // 错误类型的 panic
            igo.HandleError(c, igo.WrapError(igo.ErrCodeSystemError, "系统异常", err))
        } else {
            // 其他类型的 panic
            igo.HandleError(c, igo.NewError(igo.ErrCodeSystemError, "未知错误"))
        }
    })
}
```

### 优雅错误处理

```go
func SafeHandler(handler gin.HandlerFunc) gin.HandlerFunc {
    return func(c *gin.Context) {
        defer func() {
            if r := recover(); r != nil {
                log.Error("Handler panic",
                    log.Any("panic", r),
                    log.String("stack", string(debug.Stack())),
                )

                igo.HandleError(c, igo.NewError(igo.ErrCodeSystemError, "系统异常"))
            }
        }()

        handler(c)
    }
}
```

## 自定义错误

### 定义业务错误

```go
// 用户相关错误
var (
    ErrUserNotFound     = igo.NewError(igo.ErrCodeNotFoundError, "用户不存在")
    ErrUserAlreadyExists = igo.NewError(igo.ErrCodeConflictError, "用户已存在")
    ErrUserDisabled     = igo.NewError(igo.ErrCodeForbiddenError, "用户已被禁用")
)

// 订单相关错误
var (
    ErrOrderNotFound    = igo.NewError(igo.ErrCodeNotFoundError, "订单不存在")
    ErrOrderCancelled   = igo.NewError(igo.ErrCodeConflictError, "订单已取消")
    ErrOrderExpired     = igo.NewError(igo.ErrCodeConflictError, "订单已过期")
)
```

### 错误工厂函数

```go
// 创建用户不存在错误
func NewUserNotFoundError(userID string) error {
    return igo.NewErrorWithDetails(
        igo.ErrCodeNotFoundError,
        "用户不存在",
        map[string]interface{}{
            "user_id": userID,
        },
    )
}

// 创建参数验证错误
func NewValidationError(field, value, reason string) error {
    return igo.NewErrorWithDetails(
        igo.ErrCodeValidationError,
        "参数验证失败",
        map[string]interface{}{
            "field":  field,
            "value":  value,
            "reason": reason,
        },
    )
}
```

## 错误监控

### 错误统计

```go
type ErrorStats struct {
    Count map[int]int64
    mu    sync.RWMutex
}

var errorStats = &ErrorStats{
    Count: make(map[int]int64),
}

func RecordError(err error) {
    if appErr, ok := err.(*igo.AppError); ok {
        errorStats.mu.Lock()
        errorStats.Count[appErr.Code]++
        errorStats.mu.Unlock()
    }
}

func GetErrorStats() map[int]int64 {
    errorStats.mu.RLock()
    defer errorStats.mu.RUnlock()

    stats := make(map[int]int64)
    for code, count := range errorStats.Count {
        stats[code] = count
    }
    return stats
}
```

### 错误告警

```go
func CheckErrorRate() {
    stats := GetErrorStats()
    total := int64(0)
    for _, count := range stats {
        total += count
    }

    // 如果错误率超过阈值，发送告警
    if total > 1000 {
        log.Error("错误率过高",
            log.Int64("total_errors", total),
            log.Any("error_stats", stats),
        )
        // 发送告警通知
        sendAlert("错误率过高", stats)
    }
}
```

## 最佳实践

### 1. 错误分类

- 系统错误：记录详细日志，返回通用错误信息
- 业务错误：记录业务上下文，返回具体错误信息
- 参数错误：记录参数信息，返回验证失败信息

### 2. 错误信息

- 使用清晰、具体的错误消息
- 避免暴露敏感信息
- 提供有用的错误详情

### 3. 错误处理

- 在合适的层级处理错误
- 避免错误信息丢失
- 提供错误恢复机制

### 4. 错误监控

- 记录错误统计信息
- 设置错误率告警
- 定期分析错误模式

### 5. 错误测试

- 编写错误处理测试
- 验证错误响应格式
- 测试错误恢复机制

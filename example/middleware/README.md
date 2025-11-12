# 中间件示例说明

## 为什么中间件不在框架中？

IGo 遵循 **"简洁、边界清晰、不干预业务"** 的设计理念。中间件属于业务层关注点，不应作为脚手架的核心功能。

### IGo 的职责定位

**框架应该做的**（已提供）：
- ✅ 组件初始化和协调
- ✅ 配置管理（含热重载）
- ✅ 生命周期管理
- ✅ 基础设施封装（DB、Cache、Log）
- ✅ 优雅关闭机制

**框架不应该做的**（交给业务层）：
- ❌ 业务中间件（CORS、日志、认证等）
- ❌ 业务逻辑处理
- ❌ 路由定义
- ❌ 请求验证

### 为什么这样设计？

1. **灵活性**：每个项目的中间件需求不同，强制提供会限制灵活性
2. **简洁性**：保持框架核心简洁，避免功能膨胀
3. **可控性**：业务层完全掌控中间件的实现和行为
4. **可维护性**：职责清晰，边界明确，便于维护

## 如何使用

本目录提供的中间件仅作为**参考示例**，你可以：

1. **直接使用**（适合快速开发）
```go
import "github.com/aichy126/igo/example/middleware"

func main() {
    app, _ := igo.NewApp("")

    // 使用示例中间件
    app.Web.Router.Use(middleware.ErrorHandler())
    app.Web.Router.Use(middleware.CORSMiddleware())

    app.RunWithGracefulShutdown()
}
```

2. **根据需求修改**（推荐）
```go
// 在你的项目中创建自己的中间件
package middleware

func CustomCORS() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 你的业务逻辑
        c.Header("Access-Control-Allow-Origin", "https://yourdomain.com")
        // ...
    }
}
```

3. **完全自定义**
```go
// 使用第三方中间件库或完全自己实现
import "github.com/gin-contrib/cors"

app.Web.Router.Use(cors.Default())
```

## 示例中间件说明

本目录提供的中间件示例包括：

### 1. ErrorHandler
统一的 panic 恢复和错误处理

**使用场景**：捕获未处理的 panic，返回友好的错误信息

### 2. CORSMiddleware
跨域资源共享（CORS）支持

**使用场景**：前后端分离项目，需要处理跨域请求

### 3. LoggerMiddleware
HTTP 请求日志记录

**使用场景**：记录所有 HTTP 请求的详细信息

### 4. TimeoutMiddleware
请求超时控制

**使用场景**：防止长时间运行的请求占用资源

### 5. SecurityMiddleware
安全相关的 HTTP 头设置

**使用场景**：增强 Web 应用的安全性

## 常见中间件需求

以下是业务开发中常见的中间件需求，可参考实现：

### 认证中间件
```go
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.JSON(401, gin.H{"error": "未授权"})
            c.Abort()
            return
        }
        // 验证 token...
        c.Next()
    }
}
```

### 限流中间件
```go
import "golang.org/x/time/rate"

func RateLimitMiddleware(r rate.Limit, b int) gin.HandlerFunc {
    limiter := rate.NewLimiter(r, b)
    return func(c *gin.Context) {
        if !limiter.Allow() {
            c.JSON(429, gin.H{"error": "请求过于频繁"})
            c.Abort()
            return
        }
        c.Next()
    }
}
```

### 请求ID中间件
```go
func RequestIDMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        requestID := uuid.New().String()
        c.Set("requestID", requestID)
        c.Header("X-Request-ID", requestID)
        c.Next()
    }
}
```

## 推荐的第三方中间件库

如果不想自己实现，可以使用这些成熟的第三方库：

- **gin-contrib/cors** - CORS 支持
- **gin-contrib/sessions** - Session 管理
- **gin-contrib/gzip** - Gzip 压缩
- **ulule/limiter** - 限流
- **casbin** - 权限控制

## 总结

- IGo 专注于脚手架核心功能，不提供业务中间件
- 本目录的中间件仅作为参考示例
- 业务层应根据自己的需求实现或选择合适的中间件

这样的设计让 IGo 保持简洁优雅，同时给业务层足够的自由度。

package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/aichy126/igo/log"
	"github.com/gin-gonic/gin"
)

// 以下中间件仅供参考，业务层可根据自己的需求自定义实现

// ErrorHandler 错误处理中间件示例
func ErrorHandler() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			log.Error("Panic recovered", log.Any("error", err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    1000,
				"message": "内部服务器错误",
			})
		} else {
			log.Error("未知错误", log.Any("error", recovered))
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    1000,
				"message": "内部服务器错误",
			})
		}
		c.Abort()
	})
}

// CORSMiddleware CORS中间件示例
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// LoggerMiddleware 日志中间件示例
func LoggerMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		log.Info("HTTP请求",
			log.Any("method", param.Method),
			log.Any("path", param.Path),
			log.Any("status", param.StatusCode),
			log.Any("latency", param.Latency),
			log.Any("client_ip", param.ClientIP),
			log.Any("user_agent", param.Request.UserAgent()),
		)
		return ""
	})
}

// TimeoutMiddleware 超时中间件示例
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		done := make(chan struct{})
		go func() {
			c.Next()
			done <- struct{}{}
		}()

		select {
		case <-done:
			return
		case <-ctx.Done():
			log.Warn("请求超时", log.Any("path", c.Request.URL.Path))
			c.JSON(http.StatusRequestTimeout, gin.H{
				"code":    1000,
				"message": "请求超时",
			})
			c.Abort()
			return
		}
	}
}

// SecurityMiddleware 安全中间件示例
func SecurityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置安全相关的HTTP头
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		c.Next()
	}
}

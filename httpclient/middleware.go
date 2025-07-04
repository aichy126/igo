package httpclient

import (
	"context"
	"net/http"
	"time"

	igocontext "github.com/aichy126/igo/context"
	"github.com/aichy126/igo/log"
)

// Middleware 中间件接口
type Middleware interface {
	Process(ctx igocontext.IContext, req *http.Request, next NextHandler) (*http.Response, error)
}

// NextHandler 下一个处理器
type NextHandler func(ctx igocontext.IContext, req *http.Request) (*http.Response, error)

// MiddlewareFunc 中间件函数类型
type MiddlewareFunc func(ctx igocontext.IContext, req *http.Request, next NextHandler) (*http.Response, error)

// Process 实现 Middleware 接口
func (f MiddlewareFunc) Process(ctx igocontext.IContext, req *http.Request, next NextHandler) (*http.Response, error) {
	return f(ctx, req, next)
}

// Chain 中间件链
type Chain struct {
	middlewares []Middleware
}

// NewChain 创建中间件链
func NewChain(middlewares ...Middleware) *Chain {
	return &Chain{
		middlewares: middlewares,
	}
}

// Add 添加中间件
func (c *Chain) Add(middleware Middleware) *Chain {
	c.middlewares = append(c.middlewares, middleware)
	return c
}

// AddFunc 添加中间件函数
func (c *Chain) AddFunc(fn MiddlewareFunc) *Chain {
	return c.Add(fn)
}

// Execute 执行中间件链
func (c *Chain) Execute(ctx igocontext.IContext, req *http.Request, final NextHandler) (*http.Response, error) {
	if len(c.middlewares) == 0 {
		return final(ctx, req)
	}

	var next NextHandler
	next = func(ctx igocontext.IContext, req *http.Request) (*http.Response, error) {
		if len(c.middlewares) == 0 {
			return final(ctx, req)
		}
		middleware := c.middlewares[0]
		c.middlewares = c.middlewares[1:]
		return middleware.Process(ctx, req, next)
	}

	return next(ctx, req)
}

// 常用中间件

// LoggingMiddleware 日志中间件
func LoggingMiddleware() MiddlewareFunc {
	return func(ctx igocontext.IContext, req *http.Request, next NextHandler) (*http.Response, error) {
		start := time.Now()

		// 记录请求信息
		ctx.LogInfo("HTTP请求开始",
			log.String("method", req.Method),
			log.String("url", req.URL.String()),
			log.String("user_agent", req.UserAgent()),
		)

		// 执行请求
		resp, err := next(ctx, req)

		// 记录响应信息
		duration := time.Since(start)
		if err != nil {
			ctx.LogError("HTTP请求失败",
				log.String("method", req.Method),
				log.String("url", req.URL.String()),
				log.Duration("duration", duration),
				log.String("error", err.Error()),
			)
		} else {
			ctx.LogInfo("HTTP请求完成",
				log.String("method", req.Method),
				log.String("url", req.URL.String()),
				log.Int("status_code", resp.StatusCode),
				log.Duration("duration", duration),
			)
		}

		return resp, err
	}
}

// RetryMiddleware 重试中间件
func RetryMiddleware(maxRetries int, backoff time.Duration) MiddlewareFunc {
	return func(ctx igocontext.IContext, req *http.Request, next NextHandler) (*http.Response, error) {
		var lastErr error
		for i := 0; i <= maxRetries; i++ {
			resp, err := next(ctx, req)
			if err == nil {
				return resp, nil
			}

			lastErr = err
			if i < maxRetries {
				ctx.LogInfo("HTTP请求重试",
					log.String("method", req.Method),
					log.String("url", req.URL.String()),
					log.Int("retry", i+1),
					log.String("error", err.Error()),
				)
				time.Sleep(backoff * time.Duration(i+1))
			}
		}
		return nil, lastErr
	}
}

// TimeoutMiddleware 超时中间件
func TimeoutMiddleware(timeout time.Duration) MiddlewareFunc {
	return func(ctx igocontext.IContext, req *http.Request, next NextHandler) (*http.Response, error) {
		timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// 创建带超时的请求
		reqWithTimeout := req.WithContext(timeoutCtx)
		return next(ctx, reqWithTimeout)
	}
}

// HeaderMiddleware 请求头中间件
func HeaderMiddleware(headers map[string]string) MiddlewareFunc {
	return func(ctx igocontext.IContext, req *http.Request, next NextHandler) (*http.Response, error) {
		for key, value := range headers {
			req.Header.Set(key, value)
		}
		return next(ctx, req)
	}
}

// AuthMiddleware 认证中间件
func AuthMiddleware(authType string, credentials string) MiddlewareFunc {
	return func(ctx igocontext.IContext, req *http.Request, next NextHandler) (*http.Response, error) {
		switch authType {
		case "Bearer":
			req.Header.Set("Authorization", "Bearer "+credentials)
		case "Basic":
			req.Header.Set("Authorization", "Basic "+credentials)
		case "APIKey":
			req.Header.Set("X-API-Key", credentials)
		}
		return next(ctx, req)
	}
}

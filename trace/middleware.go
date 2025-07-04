package trace

import (
	"context"
	"time"
)

// TraceMiddleware 追踪中间件
type TraceMiddleware struct {
	tracer Tracer
}

// NewTraceMiddleware 创建追踪中间件
func NewTraceMiddleware(tracer Tracer) *TraceMiddleware {
	return &TraceMiddleware{
		tracer: tracer,
	}
}

// TraceFunc 追踪函数执行
func (tm *TraceMiddleware) TraceFunc(ctx context.Context, name string, fn func(context.Context) error) error {
	spanCtx, span := tm.tracer.StartSpan(ctx, name)
	if span == nil {
		return fn(ctx)
	}

	defer func() {
		EndSpan(span, nil)
	}()

	return fn(spanCtx)
}

// TraceFuncWithResult 追踪带返回值的函数执行
func (tm *TraceMiddleware) TraceFuncWithResult(ctx context.Context, name string, fn func(context.Context) (interface{}, error)) (interface{}, error) {
	spanCtx, span := tm.tracer.StartSpan(ctx, name)
	if span == nil {
		return fn(ctx)
	}

	defer func() {
		EndSpan(span, nil)
	}()

	return fn(spanCtx)
}

// TraceDB 追踪数据库操作
func (tm *TraceMiddleware) TraceDB(ctx context.Context, operation string, query string, args ...interface{}) (context.Context, *Span) {
	spanCtx, span := tm.tracer.StartSpan(ctx, "DB "+operation,
		WithAttributes(map[string]string{
			"db.operation": operation,
			"db.query":     query,
		}),
	)

	if span != nil {
		AddEvent(span, "db.start", map[string]string{
			"operation": operation,
			"query":     query,
		})
	}

	return spanCtx, span
}

// TraceHTTP 追踪HTTP请求
func (tm *TraceMiddleware) TraceHTTP(ctx context.Context, method, url string) (context.Context, *Span) {
	spanCtx, span := tm.tracer.StartSpan(ctx, "HTTP "+method,
		WithAttributes(map[string]string{
			"http.method": method,
			"http.url":    url,
		}),
	)

	return spanCtx, span
}

// TraceCache 追踪缓存操作
func (tm *TraceMiddleware) TraceCache(ctx context.Context, operation, key string) (context.Context, *Span) {
	spanCtx, span := tm.tracer.StartSpan(ctx, "Cache "+operation,
		WithAttributes(map[string]string{
			"cache.operation": operation,
			"cache.key":       key,
		}),
	)

	return spanCtx, span
}

// TraceExternal 追踪外部服务调用
func (tm *TraceMiddleware) TraceExternal(ctx context.Context, service, operation string) (context.Context, *Span) {
	spanCtx, span := tm.tracer.StartSpan(ctx, service+" "+operation,
		WithAttributes(map[string]string{
			"external.service":   service,
			"external.operation": operation,
		}),
	)

	return spanCtx, span
}

// TraceTiming 追踪函数执行时间
func TraceTiming(ctx context.Context, name string, fn func(context.Context) error) error {
	start := time.Now()

	spanCtx, span := GlobalTracer.StartSpan(ctx, name)
	if span == nil {
		return fn(ctx)
	}

	defer func() {
		duration := time.Since(start)
		AddAttribute(span, "duration_ms", duration.String())
		EndSpan(span, nil)
	}()

	return fn(spanCtx)
}

// TraceTimingWithResult 追踪带返回值的函数执行时间
func TraceTimingWithResult[T any](ctx context.Context, name string, fn func(context.Context) (T, error)) (T, error) {
	start := time.Now()

	spanCtx, span := GlobalTracer.StartSpan(ctx, name)
	if span == nil {
		return fn(ctx)
	}

	defer func() {
		duration := time.Since(start)
		AddAttribute(span, "duration_ms", duration.String())
		EndSpan(span, nil)
	}()

	return fn(spanCtx)
}

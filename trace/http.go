package trace

import (
	"context"
	"net/http"
)

// HTTPHeaderCarrierImpl HTTP header载体实现
type HTTPHeaderCarrierImpl http.Header

// Get 获取header值
func (c HTTPHeaderCarrierImpl) Get(key string) string {
	return http.Header(c).Get(key)
}

// Set 设置header值
func (c HTTPHeaderCarrierImpl) Set(key, value string) {
	http.Header(c).Set(key, value)
}

// GinHeaderCarrier Gin header载体实现
type GinHeaderCarrier struct {
	headers map[string]string
}

// NewGinHeaderCarrier 创建Gin header载体
func NewGinHeaderCarrier() *GinHeaderCarrier {
	return &GinHeaderCarrier{
		headers: make(map[string]string),
	}
}

// Get 获取header值
func (c *GinHeaderCarrier) Get(key string) string {
	return c.headers[key]
}

// Set 设置header值
func (c *GinHeaderCarrier) Set(key, value string) {
	c.headers[key] = value
}

// Headers 获取所有headers
func (c *GinHeaderCarrier) Headers() map[string]string {
	return c.headers
}

// ExtractFromHTTPRequest 从HTTP请求提取追踪信息
func ExtractFromHTTPRequest(req *http.Request) (context.Context, error) {
	carrier := HTTPHeaderCarrierImpl(req.Header)
	return GlobalTracer.Extract(req.Context(), carrier)
}

// InjectToHTTPRequest 将追踪信息注入HTTP请求
func InjectToHTTPRequest(ctx context.Context, req *http.Request) error {
	carrier := HTTPHeaderCarrierImpl(req.Header)
	return GlobalTracer.Inject(ctx, carrier)
}

// ExtractFromHTTPResponse 从HTTP响应提取追踪信息
func ExtractFromHTTPResponse(resp *http.Response) (context.Context, error) {
	carrier := HTTPHeaderCarrierImpl(resp.Header)
	return GlobalTracer.Extract(context.Background(), carrier)
}

// InjectToHTTPResponse 将追踪信息注入HTTP响应
func InjectToHTTPResponse(ctx context.Context, resp *http.Response) error {
	carrier := HTTPHeaderCarrierImpl(resp.Header)
	return GlobalTracer.Inject(ctx, carrier)
}

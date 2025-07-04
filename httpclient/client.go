package httpclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	igocontext "github.com/aichy126/igo/context"
)

// Config HTTP客户端配置
type Config struct {
	Timeout         time.Duration
	ConnectTimeout  time.Duration
	MaxRetries      int
	RetryDelay      time.Duration
	UserAgent       string
	FollowRedirects bool
	EnableCookies   bool
	Proxy           func(*http.Request) (*url.URL, error)
	TLSConfig       *tls.Config
	Transport       http.RoundTripper
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Timeout:         30 * time.Second,
		ConnectTimeout:  10 * time.Second,
		MaxRetries:      0,
		RetryDelay:      time.Second,
		UserAgent:       "IGo-HTTP-Client/1.0",
		FollowRedirects: true,
		EnableCookies:   false,
	}
}

// Request HTTP请求
type Request struct {
	Method      string
	URL         string
	Headers     http.Header
	QueryParams url.Values
	Body        interface{}
	Timeout     time.Duration
	Retries     int
}

// NewRequest 创建新请求
func NewRequest(method, rawurl string) *Request {
	return &Request{
		Method:      strings.ToUpper(method),
		URL:         rawurl,
		Headers:     make(http.Header),
		QueryParams: url.Values{},
	}
}

// SetHeader 设置请求头
func (r *Request) SetHeader(key, value string) *Request {
	r.Headers.Set(key, value)
	return r
}

// SetHeaders 批量设置请求头
func (r *Request) SetHeaders(headers map[string]string) *Request {
	for k, v := range headers {
		r.Headers.Set(k, v)
	}
	return r
}

// AddQuery 添加查询参数
func (r *Request) AddQuery(key, value string) *Request {
	r.QueryParams.Add(key, value)
	return r
}

// SetQuery 设置查询参数
func (r *Request) SetQuery(key, value string) *Request {
	r.QueryParams.Set(key, value)
	return r
}

// SetBody 设置请求体
func (r *Request) SetBody(body interface{}) *Request {
	r.Body = body
	return r
}

// SetJSONBody 设置JSON请求体
func (r *Request) SetJSONBody(body interface{}) *Request {
	r.Headers.Set("Content-Type", "application/json")
	r.Body = body
	return r
}

// SetFormBody 设置表单请求体
func (r *Request) SetFormBody(data url.Values) *Request {
	r.Headers.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Body = data
	return r
}

// SetTimeout 设置超时
func (r *Request) SetTimeout(timeout time.Duration) *Request {
	r.Timeout = timeout
	return r
}

// SetRetries 设置重试次数
func (r *Request) SetRetries(retries int) *Request {
	r.Retries = retries
	return r
}

// buildURL 构建完整URL
func (r *Request) buildURL() string {
	if len(r.QueryParams) == 0 {
		return r.URL
	}
	return r.URL + "?" + r.QueryParams.Encode()
}

// buildBody 构建请求体
func (r *Request) buildBody() (io.Reader, error) {
	if r.Body == nil {
		return nil, nil
	}

	switch v := r.Body.(type) {
	case io.Reader:
		return v, nil
	case []byte:
		return bytes.NewReader(v), nil
	case string:
		return strings.NewReader(v), nil
	case url.Values:
		return strings.NewReader(v.Encode()), nil
	default:
		// 尝试JSON序列化
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		return bytes.NewReader(data), nil
	}
}

// Response HTTP响应
type Response struct {
	*http.Response
	Body []byte
}

// StatusCode 获取状态码
func (r *Response) StatusCode() int {
	return r.Response.StatusCode
}

// IsSuccess 是否成功
func (r *Response) IsSuccess() bool {
	return r.StatusCode() >= 200 && r.StatusCode() < 300
}

// IsClientError 是否客户端错误
func (r *Response) IsClientError() bool {
	return r.StatusCode() >= 400 && r.StatusCode() < 500
}

// IsServerError 是否服务器错误
func (r *Response) IsServerError() bool {
	return r.StatusCode() >= 500
}

// JSON 解析JSON响应
func (r *Response) JSON(target interface{}) error {
	return json.Unmarshal(r.Body, target)
}

// String 获取字符串响应
func (r *Response) String() string {
	return string(r.Body)
}

// Bytes 获取字节响应
func (r *Response) Bytes() []byte {
	return r.Body
}

// Error 错误信息
func (r *Response) Error() string {
	if r.IsSuccess() {
		return ""
	}
	return fmt.Sprintf("HTTP %d: %s", r.StatusCode(), r.String())
}

// Client HTTP客户端
type Client struct {
	config *Config
	client *http.Client
	chain  *Chain
	mu     sync.RWMutex
}

// NewClient 创建新客户端
func NewClient(config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	client := &Client{
		config: config,
		chain:  NewChain(),
	}

	client.initHTTPClient()
	return client
}

// initHTTPClient 初始化HTTP客户端
func (c *Client) initHTTPClient() {
	transport := c.createTransport()

	var jar http.CookieJar
	if c.config.EnableCookies {
		jar, _ = cookiejar.New(nil)
	}

	c.client = &http.Client{
		Transport:     transport,
		Jar:           jar,
		CheckRedirect: c.createRedirectPolicy(),
	}
}

// createTransport 创建传输层
func (c *Client) createTransport() http.RoundTripper {
	if c.config.Transport != nil {
		return c.config.Transport
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   c.config.ConnectTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		DisableCompression:  false,
	}

	if c.config.TLSConfig != nil {
		transport.TLSClientConfig = c.config.TLSConfig
	}

	if c.config.Proxy != nil {
		transport.Proxy = c.config.Proxy
	}

	return transport
}

// createRedirectPolicy 创建重定向策略
func (c *Client) createRedirectPolicy() func(req *http.Request, via []*http.Request) error {
	if !c.config.FollowRedirects {
		return func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	return nil
}

// Use 使用中间件
func (c *Client) Use(middleware Middleware) *Client {
	c.chain.Add(middleware)
	return c
}

// UseFunc 使用中间件函数
func (c *Client) UseFunc(fn MiddlewareFunc) *Client {
	c.chain.AddFunc(fn)
	return c
}

// Do 执行请求
func (c *Client) Do(ctx igocontext.IContext, req *Request) (*Response, error) {
	httpReq, err := c.buildHTTPRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	final := func(ctx igocontext.IContext, req *http.Request) (*http.Response, error) {
		return c.client.Do(req)
	}

	httpResp, err := c.chain.Execute(ctx, httpReq, final)
	if err != nil {
		return nil, err
	}

	return c.buildResponse(httpResp)
}

// buildHTTPRequest 构建HTTP请求
func (c *Client) buildHTTPRequest(ctx igocontext.IContext, req *Request) (*http.Request, error) {
	body, err := req.buildBody()
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest(req.Method, req.buildURL(), body)
	if err != nil {
		return nil, err
	}

	// 设置请求头
	for k, v := range req.Headers {
		httpReq.Header[k] = v
	}

	// 设置User-Agent
	if httpReq.Header.Get("User-Agent") == "" {
		httpReq.Header.Set("User-Agent", c.config.UserAgent)
	}

	// 添加trace信息
	if ctx != nil {
		if traceId := getTraceIdCompat(ctx); traceId != "" {
			httpReq.Header.Set("X-Trace-ID", traceId)
		}
	}

	// 设置超时
	if req.Timeout > 0 {
		timeoutCtx, cancel := context.WithTimeout(ctx, req.Timeout)
		defer cancel()
		httpReq = httpReq.WithContext(timeoutCtx)
	}

	return httpReq, nil
}

// buildResponse 构建响应
func (c *Client) buildResponse(httpResp *http.Response) (*Response, error) {
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	return &Response{
		Response: httpResp,
		Body:     body,
	}, nil
}

// 便捷方法

// Get 执行GET请求
func (c *Client) Get(ctx igocontext.IContext, url string) (*Response, error) {
	req := NewRequest("GET", url)
	return c.Do(ctx, req)
}

// Post 执行POST请求
func (c *Client) Post(ctx igocontext.IContext, url string, body interface{}) (*Response, error) {
	req := NewRequest("POST", url).SetBody(body)
	return c.Do(ctx, req)
}

// Put 执行PUT请求
func (c *Client) Put(ctx igocontext.IContext, url string, body interface{}) (*Response, error) {
	req := NewRequest("PUT", url).SetBody(body)
	return c.Do(ctx, req)
}

// Delete 执行DELETE请求
func (c *Client) Delete(ctx igocontext.IContext, url string) (*Response, error) {
	req := NewRequest("DELETE", url)
	return c.Do(ctx, req)
}

// GetJSON 获取JSON数据
func (c *Client) GetJSON(ctx igocontext.IContext, url string, target interface{}) error {
	resp, err := c.Get(ctx, url)
	if err != nil {
		return err
	}
	return resp.JSON(target)
}

// PostJSON 发送JSON数据
func (c *Client) PostJSON(ctx igocontext.IContext, url string, data interface{}, target interface{}) error {
	req := NewRequest("POST", url).SetJSONBody(data)
	resp, err := c.Do(ctx, req)
	if err != nil {
		return err
	}
	if target != nil {
		return resp.JSON(target)
	}
	return nil
}

// PostForm 发送表单数据
func (c *Client) PostForm(ctx igocontext.IContext, url string, data url.Values, target interface{}) error {
	req := NewRequest("POST", url).SetFormBody(data)
	resp, err := c.Do(ctx, req)
	if err != nil {
		return err
	}
	if target != nil {
		return resp.JSON(target)
	}
	return nil
}

// 链式配置方法

// WithTimeout 设置超时
func (c *Client) WithTimeout(timeout time.Duration) *Client {
	c.UseFunc(TimeoutMiddleware(timeout))
	return c
}

// WithRetry 设置重试
func (c *Client) WithRetry(maxRetries int, backoff time.Duration) *Client {
	c.UseFunc(RetryMiddleware(maxRetries, backoff))
	return c
}

// WithLogging 启用日志
func (c *Client) WithLogging() *Client {
	c.UseFunc(LoggingMiddleware())
	return c
}

// WithHeaders 设置请求头
func (c *Client) WithHeaders(headers map[string]string) *Client {
	c.UseFunc(HeaderMiddleware(headers))
	return c
}

// WithAuth 设置认证
func (c *Client) WithAuth(authType, credentials string) *Client {
	c.UseFunc(AuthMiddleware(authType, credentials))
	return c
}

// getTraceIdCompat 兼容 string/[]string 类型的 traceId
func getTraceIdCompat(ctx igocontext.IContext) string {
	if ctx == nil {
		return ""
	}
	v, ok := ctx.Get("traceId")
	if !ok || v == nil {
		return ""
	}
	switch vv := v.(type) {
	case string:
		return vv
	case []string:
		if len(vv) > 0 {
			return vv[0]
		}
	}
	return fmt.Sprintf("%v", v)
}

package httpclient

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aichy126/igo/ictx"
	"github.com/aichy126/igo/ilog"
)

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries  int           // 最大重试次数
	InitialWait time.Duration // 初始等待时间
	MaxWait     time.Duration // 最大等待时间
	BackoffRate float64       // 退避倍率
}

// Client 简化的HTTP客户端
type Client struct {
	client        *http.Client
	baseURL       string
	retryConfig   *RetryConfig
	userAgent     string
	debug         bool
	defaultHeaders http.Header
}

// New 创建新的HTTP客户端
func New() *Client {
	return &Client{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		retryConfig:    nil, // 默认不启用重试
		debug:          false,
		defaultHeaders: make(http.Header),
	}
}

// NewWithTimeout 创建带超时的HTTP客户端
func NewWithTimeout(timeout time.Duration) *Client {
	return &Client{
		client: &http.Client{
			Timeout: timeout,
		},
		retryConfig:    nil, // 默认不启用重试
		debug:          false,
		defaultHeaders: make(http.Header),
	}
}

// SetDefaultTimeout 设置默认超时时间
func (c *Client) SetDefaultTimeout(timeout time.Duration) *Client {
	c.client.Timeout = timeout
	return c
}

// SetDefaultRetries 设置默认重试次数
func (c *Client) SetDefaultRetries(maxRetries int) *Client {
	c.retryConfig = &RetryConfig{
		MaxRetries:  maxRetries,
		InitialWait: 100 * time.Millisecond,
		MaxWait:     5 * time.Second,
		BackoffRate: 2.0,
	}
	return c
}

// SetUserAgent 设置User-Agent
func (c *Client) SetUserAgent(userAgent string) *Client {
	c.userAgent = userAgent
	return c
}

// Debug 设置调试模式
func (c *Client) Debug(debug bool) *Client {
	c.debug = debug
	return c
}

// SetHeader 设置默认请求头
func (c *Client) SetHeader(key, value string) *Client {
	c.defaultHeaders.Set(key, value)
	return c
}

// SetHeaders 批量设置默认请求头
func (c *Client) SetHeaders(headers map[string]string) *Client {
	for key, value := range headers {
		c.defaultHeaders.Set(key, value)
	}
	return c
}

// AddHeader 添加默认请求头（不覆盖已存在的）
func (c *Client) AddHeader(key, value string) *Client {
	c.defaultHeaders.Add(key, value)
	return c
}

// SetTLSClientConfig 设置TLS配置
func (c *Client) SetTLSClientConfig(config *tls.Config) *Client {
	if c.client.Transport == nil {
		c.client.Transport = &http.Transport{}
	}
	if transport, ok := c.client.Transport.(*http.Transport); ok {
		transport.TLSClientConfig = config
	}
	return c
}

// SetBaseURL 设置基础URL
func (c *Client) SetBaseURL(baseURL string) *Client {
	c.baseURL = strings.TrimSuffix(baseURL, "/")
	return c
}

// Request HTTP请求结构
type Request struct {
	Method string
	URL    string
	Body   io.Reader
	Header http.Header
}

// Response HTTP响应结构
type Response struct {
	*http.Response
	Body []byte
}

// JSON 解析响应为JSON
func (r *Response) JSON(target interface{}) error {
	return json.Unmarshal(r.Body, target)
}

// String 获取响应字符串
func (r *Response) String() string {
	return string(r.Body)
}

// Do 执行HTTP请求（核心方法，不包含重试）
func (c *Client) Do(ctx ictx.Context, req *Request) (*Response, error) {
	return c.doSingleRequest(ctx, req)
}

// DoWithRetry 执行HTTP请求（带重试功能）
func (c *Client) DoWithRetry(ctx ictx.Context, req *Request) (*Response, error) {
	// 如果没有配置重试，直接执行单次请求
	if c.retryConfig == nil || c.retryConfig.MaxRetries <= 0 {
		return c.doSingleRequest(ctx, req)
	}
	
	var lastErr error
	
	// 执行重试逻辑
	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		// 如果不是第一次尝试，等待一段时间
		if attempt > 0 {
			waitTime := c.calculateBackoff(attempt)
			if ctx != nil && c.debug {
				ctx.LogInfo("HTTP请求重试",
					log.Int("attempt", attempt+1),
					log.String("wait", waitTime.String()),
					log.String("lastError", lastErr.Error()))
			}
			time.Sleep(waitTime)
		}
		
		// 执行单次请求
		resp, err := c.doSingleRequest(ctx, req)
		if err == nil {
			// 成功则直接返回
			if attempt > 0 && ctx != nil && c.debug {
				ctx.LogInfo("HTTP请求重试成功", log.Int("totalAttempts", attempt+1))
			}
			return resp, nil
		}
		
		// 检查是否应该重试
		if !c.shouldRetry(err, resp) {
			return resp, err
		}
		
		lastErr = err
	}
	
	// 所有重试都失败
	if ctx != nil {
		ctx.LogError("HTTP请求最终失败",
			log.Int("totalAttempts", c.retryConfig.MaxRetries+1),
			log.String("finalError", lastErr.Error()))
	}
	return nil, lastErr
}

// doSingleRequest 执行单次HTTP请求
func (c *Client) doSingleRequest(ctx ictx.Context, req *Request) (*Response, error) {
	// 构建完整URL
	fullURL := req.URL
	if c.baseURL != "" && !strings.HasPrefix(req.URL, "http") {
		fullURL = c.baseURL + "/" + strings.TrimPrefix(req.URL, "/")
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequest(req.Method, fullURL, req.Body)
	if err != nil {
		return nil, err
	}

	// 首先设置默认请求头
	for key, values := range c.defaultHeaders {
		for _, value := range values {
			httpReq.Header.Add(key, value)
		}
	}

	// 然后设置请求特定的请求头（会覆盖默认的）
	if req.Header != nil {
		for key, values := range req.Header {
			httpReq.Header.Del(key) // 先删除默认的
			for _, value := range values {
				httpReq.Header.Add(key, value)
			}
		}
	}
	
	// 设置User-Agent（如果没有在header中指定）
	if c.userAgent != "" && httpReq.Header.Get("User-Agent") == "" {
		httpReq.Header.Set("User-Agent", c.userAgent)
	}

	// 自动添加traceId到请求头（这是保留httpclient的核心原因）
	if ctx != nil {
		if traceId := ctx.GetString("traceId"); traceId != "" {
			httpReq.Header.Set("X-Trace-ID", traceId)
		}
	}

	// 设置Content-Type（如果没有设置且有body）
	if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// 记录请求日志（通过ictx自动带traceId）
	if ctx != nil && c.debug {
		ctx.LogInfo("HTTP请求开始",
			log.String("method", req.Method),
			log.String("url", fullURL))
	}

	// 执行请求
	httpResp, err := c.client.Do(httpReq)
	if err != nil {
		if ctx != nil {
			ctx.LogError("HTTP请求失败", log.String("error", err.Error()))
		}
		return nil, err
	}
	defer httpResp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	// 记录响应日志（通过ictx自动带traceId）
	if ctx != nil && c.debug {
		ctx.LogInfo("HTTP请求完成",
			log.Int("status", httpResp.StatusCode),
			log.Int("size", len(body)))
	}

	return &Response{
		Response: httpResp,
		Body:     body,
	}, nil
}

// Get 请求
func (c *Client) Get(ctx ictx.Context, url string) (*Response, error) {
	req := &Request{
		Method: "GET",
		URL:    url,
	}

	// 根据客户端配置决定是否使用重试
	if c.retryConfig != nil && c.retryConfig.MaxRetries > 0 {
		return c.DoWithRetry(ctx, req)
	}
	return c.Do(ctx, req)
}

// Post 请求
func (c *Client) Post(ctx ictx.Context, url string, body interface{}) (*Response, error) {
	var bodyReader io.Reader

	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	req := &Request{
		Method: "POST",
		URL:    url,
		Body:   bodyReader,
	}

	// 根据客户端配置决定是否使用重试
	if c.retryConfig != nil && c.retryConfig.MaxRetries > 0 {
		return c.DoWithRetry(ctx, req)
	}
	return c.Do(ctx, req)
}

// Put 请求
func (c *Client) Put(ctx ictx.Context, url string, body interface{}) (*Response, error) {
	var bodyReader io.Reader

	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	req := &Request{
		Method: "PUT",
		URL:    url,
		Body:   bodyReader,
	}

	// 根据客户端配置决定是否使用重试
	if c.retryConfig != nil && c.retryConfig.MaxRetries > 0 {
		return c.DoWithRetry(ctx, req)
	}
	return c.Do(ctx, req)
}

// Delete 请求
func (c *Client) Delete(ctx ictx.Context, url string) (*Response, error) {
	req := &Request{
		Method: "DELETE",
		URL:    url,
	}

	// 根据客户端配置决定是否使用重试
	if c.retryConfig != nil && c.retryConfig.MaxRetries > 0 {
		return c.DoWithRetry(ctx, req)
	}
	return c.Do(ctx, req)
}

// PostForm 发送表单数据
func (c *Client) PostForm(ctx ictx.Context, url string, data url.Values) (*Response, error) {
	req := &Request{
		Method: "POST",
		URL:    url,
		Body:   strings.NewReader(data.Encode()),
		Header: make(http.Header),
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// 根据客户端配置决定是否使用重试
	if c.retryConfig != nil && c.retryConfig.MaxRetries > 0 {
		return c.DoWithRetry(ctx, req)
	}
	return c.Do(ctx, req)
}

// GetBodyString 获取响应体字符串的便捷方法（支持任何HTTP方法）
func (c *Client) GetBodyString(ctx ictx.Context, method, url string, body interface{}) (string, error) {
	var bodyReader io.Reader

	if body != nil {
		switch v := body.(type) {
		case string:
			bodyReader = strings.NewReader(v)
		case []byte:
			bodyReader = bytes.NewReader(v)
		case io.Reader:
			bodyReader = v
		default:
			// 默认序列化为JSON
			jsonData, err := json.Marshal(body)
			if err != nil {
				return "", err
			}
			bodyReader = bytes.NewReader(jsonData)
		}
	}

	req := &Request{
		Method: method,
		URL:    url,
		Body:   bodyReader,
	}

	var resp *Response
	var err error

	// 根据客户端配置决定是否使用重试
	if c.retryConfig != nil && c.retryConfig.MaxRetries > 0 {
		resp, err = c.DoWithRetry(ctx, req)
	} else {
		resp, err = c.Do(ctx, req)
	}

	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.String())
	}

	return resp.String(), nil
}

// GetJSON 获取JSON数据的便捷方法
func (c *Client) GetJSON(ctx ictx.Context, url string, target interface{}) error {
	resp, err := c.Get(ctx, url)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.String())
	}

	return resp.JSON(target)
}

// PostJSON 发送JSON数据的便捷方法
func (c *Client) PostJSON(ctx ictx.Context, url string, data interface{}, target interface{}) error {
	resp, err := c.Post(ctx, url, data)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.String())
	}

	if target != nil {
		return resp.JSON(target)
	}
	return nil
}

// calculateBackoff 计算重试等待时间（指数退避）
func (c *Client) calculateBackoff(attempt int) time.Duration {
	if c.retryConfig == nil {
		return 0
	}
	
	// 指数退避算法
	waitTime := float64(c.retryConfig.InitialWait) * math.Pow(c.retryConfig.BackoffRate, float64(attempt-1))
	
	// 限制最大等待时间
	if time.Duration(waitTime) > c.retryConfig.MaxWait {
		return c.retryConfig.MaxWait
	}
	
	return time.Duration(waitTime)
}

// shouldRetry 判断是否应该重试
func (c *Client) shouldRetry(err error, resp *Response) bool {
	// 网络错误时重试
	if err != nil {
		return true
	}
	
	// 根据HTTP状态码判断是否重试
	if resp != nil {
		statusCode := resp.StatusCode
		// 只重试5xx服务器错误和429限流
		return statusCode >= 500 || statusCode == 429
	}
	
	return false
}

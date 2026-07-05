// Package httpclient 是 net/http 的轻量封装:ctx-first、JSON 便捷方法、
// 简单重试,并自动透传 igo context 的 meta header(含 traceId)到下游服务。
//
// 基本用法:
//
//	client := httpclient.New(httpclient.WithTimeout(3*time.Second))
//	var out SomeResp
//	err := client.PostJSON(ctx, url, req, &out)
//
// 简单请求可直接用包级默认客户端:
//
//	resp, err := httpclient.Get(ctx, url)
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
	"net/url"
	"strings"
	"time"

	"github.com/aichy126/igo/log"
)

// headerCarrier 由 igo 的 context.IContext 实现;用结构化接口避免包依赖
type headerCarrier interface {
	GetHeaders() http.Header
}

// Client HTTP 客户端,并发安全,建议复用(内部连接池)
type Client struct {
	hc      *http.Client
	headers http.Header // 每个请求都会附带的默认 header(如 User-Agent)
	retries int         // 网络层错误的重试次数(HTTP 状态码错误不重试)
	debug   bool        // 打印请求/响应日志(Debug 级别)
}

// Option 客户端配置项
type Option func(*Client)

// WithTimeout 设置整体请求超时(默认 10s)
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.hc.Timeout = d }
}

// WithUserAgent 设置 User-Agent
func WithUserAgent(ua string) Option {
	return func(c *Client) { c.headers.Set("User-Agent", ua) }
}

// WithHeader 设置每个请求都会附带的默认 header
func WithHeader(key, value string) Option {
	return func(c *Client) { c.headers.Set(key, value) }
}

// WithRetries 设置网络层错误(连接失败、超时等)的重试次数,HTTP 状态码错误不重试
func WithRetries(n int) Option {
	return func(c *Client) {
		if n > 0 {
			c.retries = n
		}
	}
}

// WithTransport 自定义 Transport(完全接管;与 WithProxy*/WithInsecureSkipVerify 同用时请放在它们之前)
func WithTransport(rt http.RoundTripper) Option {
	return func(c *Client) { c.hc.Transport = rt }
}

// WithProxyURL 设置固定代理地址,如 "http://127.0.0.1:7890"
func WithProxyURL(rawurl string) Option {
	return func(c *Client) {
		if t, ok := c.hc.Transport.(*http.Transport); ok {
			t.Proxy = func(_ *http.Request) (*url.URL, error) {
				return url.Parse(rawurl)
			}
		}
	}
}

// WithProxyFunc 设置动态代理函数(每次请求调用,适合代理地址从配置热读取的场景)
func WithProxyFunc(f func(*http.Request) (*url.URL, error)) Option {
	return func(c *Client) {
		if t, ok := c.hc.Transport.(*http.Transport); ok {
			t.Proxy = f
		}
	}
}

// WithInsecureSkipVerify 跳过 TLS 证书校验(自签名证书/抓包调试场景,生产慎用)
func WithInsecureSkipVerify() Option {
	return func(c *Client) {
		if t, ok := c.hc.Transport.(*http.Transport); ok {
			if t.TLSClientConfig == nil {
				t.TLSClientConfig = &tls.Config{}
			}
			t.TLSClientConfig.InsecureSkipVerify = true
		}
	}
}

// WithDebug 开启请求/响应日志
func WithDebug(b bool) Option {
	return func(c *Client) { c.debug = b }
}

// New 创建客户端(默认 10s 超时、合理的连接池配置)
func New(opts ...Option) *Client {
	c := &Client{
		hc: &http.Client{
			Timeout:   10 * time.Second,
			Transport: defaultTransport(),
		},
		headers: http.Header{},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func defaultTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 7 * time.Second, // 部分云 SLB 空闲 15s 会 reset 连接,保持较短的 keepalive
		}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       7 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

// Response HTTP 响应(body 已全部读入内存)
type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// OK 状态码是否为 2xx
func (r *Response) OK() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// String 响应体字符串
func (r *Response) String() string {
	return string(r.Body)
}

// JSON 反序列化响应体
func (r *Response) JSON(v any) error {
	return json.Unmarshal(r.Body, v)
}

// Do 发起请求并读取完整响应。body 会被完整缓冲以支持重试。
// header 参数为本次请求的附加 header,可为 nil。
func (c *Client) Do(ctx context.Context, method, rawurl string, body io.Reader, header http.Header) (*Response, error) {
	// 缓冲 body:支持重试重放,也便于 debug 输出
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("读取请求 body 失败: %w", err)
		}
	}

	var lastErr error
	attempts := c.retries + 1
	for i := 0; i < attempts; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(i) * 100 * time.Millisecond):
			}
		}
		resp, err := c.doOnce(ctx, method, rawurl, bodyBytes, header)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		// ctx 取消/超时不重试
		if ctx.Err() != nil {
			break
		}
	}
	return nil, lastErr
}

func (c *Client) doOnce(ctx context.Context, method, rawurl string, body []byte, header http.Header) (*Response, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, rawurl, reader)
	if err != nil {
		return nil, err
	}

	// 默认 header → igo context meta 透传(含 traceId) → 本次请求 header,后者覆盖前者
	for k, vs := range c.headers {
		for _, v := range vs {
			req.Header.Set(k, v)
		}
	}
	if hc, ok := ctx.(headerCarrier); ok {
		for k, vs := range hc.GetHeaders() {
			for _, v := range vs {
				req.Header.Set(k, v)
			}
		}
	}
	for k, vs := range header {
		for _, v := range vs {
			req.Header.Set(k, v)
		}
	}

	start := time.Now()
	resp, err := c.hc.Do(req)
	if err != nil {
		if c.debug {
			log.Error("httpclient 请求失败", log.String("method", method), log.String("url", rawurl), log.Any("error", err))
		}
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应 body 失败: %w", err)
	}

	if c.debug {
		log.Debug("httpclient",
			log.String("method", method),
			log.String("url", rawurl),
			log.Int("status", resp.StatusCode),
			log.Duration("cost", time.Since(start)),
			log.String("request", string(body)),
			log.String("response", string(data)),
		)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Body:       data,
	}, nil
}

// ReqOption 单次请求的配置项(目前用于附加 header)
type ReqOption func(http.Header)

// WithReqHeader 为本次请求附加一个 header
func WithReqHeader(key, value string) ReqOption {
	return func(h http.Header) { h.Set(key, value) }
}

// WithReqHeaders 为本次请求附加一组 header(map[string][]string 可直接传入)
func WithReqHeaders(headers http.Header) ReqOption {
	return func(h http.Header) {
		for k, vs := range headers {
			for _, v := range vs {
				h.Add(k, v)
			}
		}
	}
}

// buildHeader 把 contentType 和请求级 option 合成 header
func buildHeader(contentType string, opts []ReqOption) http.Header {
	header := http.Header{}
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}
	for _, opt := range opts {
		opt(header)
	}
	return header
}

// Get 发起 GET 请求
func (c *Client) Get(ctx context.Context, url string, opts ...ReqOption) (*Response, error) {
	return c.Do(ctx, http.MethodGet, url, nil, buildHeader("", opts))
}

// GetBytes 发起 GET 请求并返回响应 body(非 2xx 返回错误),适合下载文件/原始内容
func (c *Client) GetBytes(ctx context.Context, url string, opts ...ReqOption) ([]byte, error) {
	resp, err := c.Get(ctx, url, opts...)
	if err != nil {
		return nil, err
	}
	if !resp.OK() {
		return nil, fmt.Errorf("http 状态码 %d: %s", resp.StatusCode, truncate(resp.String(), 200))
	}
	return resp.Body, nil
}

// Post 发起 POST 请求
func (c *Client) Post(ctx context.Context, url string, contentType string, body io.Reader, opts ...ReqOption) (*Response, error) {
	return c.Do(ctx, http.MethodPost, url, body, buildHeader(contentType, opts))
}

// PostForm 发起表单 POST 请求
func (c *Client) PostForm(ctx context.Context, url string, form url.Values, opts ...ReqOption) (*Response, error) {
	return c.Post(ctx, url, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()), opts...)
}

// PostFormJSON 发起表单 POST 请求并把 JSON 响应反序列化到 out(out 可为 nil)
func (c *Client) PostFormJSON(ctx context.Context, url string, form url.Values, out any, opts ...ReqOption) error {
	resp, err := c.PostForm(ctx, url, form, opts...)
	if err != nil {
		return err
	}
	return decodeJSONResponse(resp, out)
}

// GetJSON 发起 GET 请求并把 JSON 响应反序列化到 out(out 可为 nil)
func (c *Client) GetJSON(ctx context.Context, url string, out any, opts ...ReqOption) error {
	resp, err := c.Get(ctx, url, opts...)
	if err != nil {
		return err
	}
	return decodeJSONResponse(resp, out)
}

// PostJSON 发起 JSON POST 请求并把 JSON 响应反序列化到 out(out 可为 nil)
func (c *Client) PostJSON(ctx context.Context, url string, in any, out any, opts ...ReqOption) error {
	body, err := marshalJSONBody(in)
	if err != nil {
		return err
	}
	resp, err := c.Post(ctx, url, "application/json", body, opts...)
	if err != nil {
		return err
	}
	return decodeJSONResponse(resp, out)
}

// PutJSON 发起 JSON PUT 请求并把 JSON 响应反序列化到 out(out 可为 nil)
func (c *Client) PutJSON(ctx context.Context, url string, in any, out any, opts ...ReqOption) error {
	body, err := marshalJSONBody(in)
	if err != nil {
		return err
	}
	resp, err := c.Do(ctx, http.MethodPut, url, body, buildHeader("application/json", opts))
	if err != nil {
		return err
	}
	return decodeJSONResponse(resp, out)
}

func marshalJSONBody(in any) (io.Reader, error) {
	if in == nil {
		return nil, nil
	}
	data, err := json.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("请求 body 序列化失败: %w", err)
	}
	return bytes.NewReader(data), nil
}

func decodeJSONResponse(resp *Response, out any) error {
	if !resp.OK() {
		return fmt.Errorf("http 状态码 %d: %s", resp.StatusCode, truncate(resp.String(), 200))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(resp.Body, out); err != nil {
		return fmt.Errorf("响应 JSON 解析失败: %w (body: %s)", err, truncate(resp.String(), 200))
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// Default 包级默认客户端(10s 超时)
var Default = New()

// Get 使用默认客户端发起 GET 请求
func Get(ctx context.Context, url string, opts ...ReqOption) (*Response, error) {
	return Default.Get(ctx, url, opts...)
}

// GetBytes 使用默认客户端发起 GET 请求并返回响应 body
func GetBytes(ctx context.Context, url string, opts ...ReqOption) ([]byte, error) {
	return Default.GetBytes(ctx, url, opts...)
}

// GetJSON 使用默认客户端发起 GET 请求并解析 JSON 响应
func GetJSON(ctx context.Context, url string, out any, opts ...ReqOption) error {
	return Default.GetJSON(ctx, url, out, opts...)
}

// PostJSON 使用默认客户端发起 JSON POST 请求
func PostJSON(ctx context.Context, url string, in, out any, opts ...ReqOption) error {
	return Default.PostJSON(ctx, url, in, out, opts...)
}

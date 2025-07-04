# httpclient 组件文档

## 简介

`httpclient` 是一个灵活、可扩展的 HTTP 客户端库，支持链式中间件、traceId 透传、超时、重试、认证、日志等常用能力，适合微服务和中后台业务开发。

- 只依赖 igocontext（github.com/aichy126/igo/context）
- 支持自定义中间件链
- 便捷的请求构建与响应处理
- 兼容 traceId 分布式链路追踪

---

## 快速开始

```go
import (
    "github.com/aichy126/igo/httpclient"
    igocontext "github.com/aichy126/igo/context"
)

ctx := igocontext.NewContext()
ctx.Set("traceId", "your-trace-id")

client := httpclient.NewClient(nil).
    WithLogging().
    WithTimeout(5 * time.Second).
    WithRetry(2, 100*time.Millisecond)

// GET 请求
resp, err := client.Get(ctx, "https://api.example.com/data")
if err != nil {
    // 错误处理
}

// 解析 JSON 响应
var data map[string]interface{}
_ = resp.JSON(&data)
```

---

## 配置说明

- `Timeout`：请求超时时间
- `ConnectTimeout`：连接超时时间
- `MaxRetries`：最大重试次数
- `RetryDelay`：重试间隔
- `UserAgent`：默认 User-Agent
- `FollowRedirects`：是否跟随重定向
- `EnableCookies`：是否启用 CookieJar
- `Proxy`：自定义代理
- `TLSConfig`：自定义 TLS 配置
- `Transport`：自定义底层 Transport

可通过 `NewClient(config)` 传入自定义配置。

---

## 请求构建

```go
req := httpclient.NewRequest("POST", "https://api.example.com/user")
    .SetHeader("X-Token", "abc")
    .SetJSONBody(map[string]string{"name": "Tom"})
    .SetTimeout(2 * time.Second)

resp, err := client.Do(ctx, req)
```

- 支持 `SetHeader`、`SetHeaders`、`SetQuery`、`AddQuery`、`SetBody`、`SetJSONBody`、`SetFormBody`、`SetTimeout`、`SetRetries` 等链式方法。

---

## 响应处理

- `resp.StatusCode()` 获取状态码
- `resp.IsSuccess()` 是否2xx
- `resp.JSON(target)` 解析JSON
- `resp.String()` 获取字符串
- `resp.Bytes()` 获取字节

---

## 中间件系统

支持链式中间件，内置：
- 日志（WithLogging）
- 超时（WithTimeout）
- 重试（WithRetry）
- 认证（WithAuth）
- 请求头（WithHeaders）

自定义中间件：
```go
client.UseFunc(func(ctx igocontext.IContext, req *http.Request, next httpclient.NextHandler) (*http.Response, error) {
    req.Header.Set("X-Request-ID", "req-12345")
    return next(ctx, req)
})
```

---

## traceId 透传

- 只需在 igocontext 中设置 traceId，httpclient 会自动将其注入 X-Trace-ID 请求头。
- 支持 string/[]string 类型。

---

## 常见用法示例

### GET/POST/PUT/DELETE
```go
client.Get(ctx, url)
client.Post(ctx, url, body)
client.Put(ctx, url, body)
client.Delete(ctx, url)
```

### JSON/表单请求
```go
client.GetJSON(ctx, url, &respObj)
client.PostJSON(ctx, url, reqObj, &respObj)
client.PostForm(ctx, url, formValues, &respObj)
```

### 链式配置
```go
client.WithTimeout(2*time.Second).WithRetry(3, time.Second).WithLogging()
```

---

## 单元测试与扩展

- 推荐使用 httptest.NewServer 进行端到端测试
- 支持自定义中间件链和请求构建
- 详见 `httpclient/example_test.go`，已覆盖所有核心能力

---

## 最佳实践

- 推荐统一通过 igocontext 传递 traceId，实现全链路追踪
- 通过中间件扩展日志、认证、限流、熔断等能力
- 结合配置中心动态调整超时、重试等参数

---

## 参考
- [Go net/http 官方文档](https://pkg.go.dev/net/http)
- [httptest 用法](https://pkg.go.dev/net/http/httptest)

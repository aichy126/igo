# Log 钩子使用指南

## 概述

IGo 的日志钩子系统允许业务层拦截特定级别的日志，并将其发送到第三方服务（如飞书、企业微信、钉钉等）。

## 特性

- ✅ 支持按日志级别过滤
- ✅ 异步执行，不阻塞日志记录
- ✅ 支持多个钩子同时工作
- ✅ 自动提取 traceId 和日志字段
- ✅ 钩子执行失败不影响日志系统

## 快速开始

### 1. 实现钩子接口

```go
package main

import (
    "github.com/aichy126/igo/log"
    "go.uber.org/zap/zapcore"
)

// FeishuHook 飞书通知钩子
type FeishuHook struct {
    WebhookURL string
}

// Levels 返回关注的日志级别
func (h *FeishuHook) Levels() []zapcore.Level {
    // 只关注 Error 和 Fatal 级别
    return []zapcore.Level{
        zapcore.ErrorLevel,
        zapcore.FatalLevel,
    }
}

// Fire 当有匹配级别的日志时触发
func (h *FeishuHook) Fire(entry *log.LogEntry) error {
    // 构建飞书消息
    message := fmt.Sprintf(
        "⚠️ 错误告警\\n级别: %s\\n消息: %s\\nTraceID: %s\\n时间: %s",
        entry.Level.String(),
        entry.Message,
        entry.TraceID,
        entry.Timestamp.Format("2006-01-02 15:04:05"),
    )

    // 发送到飞书
    return sendToFeishu(h.WebhookURL, message)
}
```

### 2. 注册钩子

```go
func main() {
    // 初始化应用
    app, err := igo.NewApp("")
    if err != nil {
        log.Fatal("初始化失败", log.Any("error", err))
    }

    // 注册飞书钩子
    feishuHook := &FeishuHook{
        WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/xxx",
    }
    log.AddHook(feishuHook)

    // 注册企业微信钩子
    wechatHook := &WechatHook{
        WebhookURL: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx",
    }
    log.AddHook(wechatHook)

    // 运行应用
    app.RunWithGracefulShutdown()
}
```

### 3. 触发钩子

```go
// 这些日志会触发钩子
log.Error("数据库连接失败", log.Any("error", err))
log.Fatal("配置文件错误", log.Any("path", configPath))

// 这些日志不会触发钩子（级别不匹配）
log.Info("用户登录成功", log.Any("userId", 123))
log.Warn("缓存未命中", log.Any("key", "user:123"))
```

## LogEntry 结构

钩子的 `Fire()` 方法接收一个 `LogEntry` 对象，包含以下字段：

```go
type LogEntry struct {
    Level     zapcore.Level          // 日志级别
    Message   string                 // 日志消息
    Fields    map[string]interface{} // 日志字段
    Timestamp time.Time              // 时间戳
    TraceID   string                 // 追踪ID（如果存在）
}
```

## 完整示例

### 飞书钩子实现

```go
package hooks

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/aichy126/igo/log"
    "go.uber.org/zap/zapcore"
)

type FeishuHook struct {
    WebhookURL string
    AppName    string
}

func (h *FeishuHook) Levels() []zapcore.Level {
    return []zapcore.Level{
        zapcore.ErrorLevel,
        zapcore.FatalLevel,
        zapcore.PanicLevel,
    }
}

func (h *FeishuHook) Fire(entry *log.LogEntry) error {
    // 构建飞书卡片消息
    card := map[string]interface{}{
        "msg_type": "interactive",
        "card": map[string]interface{}{
            "header": map[string]interface{}{
                "title": map[string]string{
                    "content": fmt.Sprintf("🚨 [%s] %s", h.AppName, entry.Level.String()),
                    "tag":     "plain_text",
                },
                "template": "red",
            },
            "elements": []interface{}{
                map[string]interface{}{
                    "tag": "div",
                    "text": map[string]string{
                        "content": fmt.Sprintf("**消息**: %s", entry.Message),
                        "tag":     "lark_md",
                    },
                },
                map[string]interface{}{
                    "tag": "div",
                    "text": map[string]string{
                        "content": fmt.Sprintf("**TraceID**: %s", entry.TraceID),
                        "tag":     "lark_md",
                    },
                },
                map[string]interface{}{
                    "tag": "div",
                    "text": map[string]string{
                        "content": fmt.Sprintf("**时间**: %s", entry.Timestamp.Format("2006-01-02 15:04:05")),
                        "tag":     "lark_md",
                    },
                },
            },
        },
    }

    // 发送HTTP请求
    body, _ := json.Marshal(card)
    resp, err := http.Post(h.WebhookURL, "application/json", bytes.NewBuffer(body))
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("飞书通知失败: %d", resp.StatusCode)
    }

    return nil
}
```

### 企业微信钩子实现

```go
package hooks

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/aichy126/igo/log"
    "go.uber.org/zap/zapcore"
)

type WechatHook struct {
    WebhookURL string
    Mentioned  []string // @的用户列表
}

func (h *WechatHook) Levels() []zapcore.Level {
    return []zapcore.Level{
        zapcore.ErrorLevel,
        zapcore.FatalLevel,
    }
}

func (h *WechatHook) Fire(entry *log.LogEntry) error {
    // 构建企业微信消息
    message := map[string]interface{}{
        "msgtype": "markdown",
        "markdown": map[string]interface{}{
            "content": fmt.Sprintf(
                "## 错误告警\n"+
                    "> 级别: <font color=\"warning\">%s</font>\n"+
                    "> 消息: %s\n"+
                    "> TraceID: %s\n"+
                    "> 时间: %s",
                entry.Level.String(),
                entry.Message,
                entry.TraceID,
                entry.Timestamp.Format("2006-01-02 15:04:05"),
            ),
            "mentioned_list": h.Mentioned,
        },
    }

    body, _ := json.Marshal(message)
    resp, err := http.Post(h.WebhookURL, "application/json", bytes.NewBuffer(body))
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    return nil
}
```

## 注意事项

1. **异步执行**: 钩子在独立的 goroutine 中执行，不会阻塞日志记录
2. **错误处理**: 钩子执行失败不会影响日志系统，会被静默处理
3. **性能考虑**: 避免在钩子中执行耗时操作，如需要可考虑使用消息队列
4. **并发安全**: 钩子的 `Fire()` 方法可能被并发调用，需要注意线程安全

## 最佳实践

1. **限流**: 在钩子中实现限流机制，避免频繁通知
2. **批量发送**: 收集一定时间内的日志，批量发送
3. **重试机制**: 网络请求失败时实现重试
4. **监控**: 监控钩子的执行状态和失败率

```go
type RateLimitedHook struct {
    inner    log.LogHook
    limiter  *rate.Limiter
}

func (h *RateLimitedHook) Levels() []zapcore.Level {
    return h.inner.Levels()
}

func (h *RateLimitedHook) Fire(entry *log.LogEntry) error {
    if !h.limiter.Allow() {
        return nil // 限流，跳过
    }
    return h.inner.Fire(entry)
}
```

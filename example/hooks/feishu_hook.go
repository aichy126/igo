package hooks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aichy126/igo/log"
	"go.uber.org/zap/zapcore"
)

// FeishuHook 飞书通知钩子示例
// 使用方式：log.AddHook(&hooks.FeishuHook{WebhookURL: "your-webhook-url"})
type FeishuHook struct {
	WebhookURL string
	AppName    string
	Enabled    bool // 开关，方便测试时禁用
}

// Levels 返回关注的日志级别
func (h *FeishuHook) Levels() []zapcore.Level {
	// 只关注 Error 和 Fatal 级别
	return []zapcore.Level{
		zapcore.ErrorLevel,
		zapcore.FatalLevel,
		zapcore.PanicLevel,
	}
}

// Fire 当有匹配级别的日志时触发
func (h *FeishuHook) Fire(entry *log.LogEntry) error {
	if !h.Enabled {
		return nil // 禁用时直接返回
	}

	// 构建飞书卡片消息
	card := h.buildFeishuCard(entry)

	// 发送HTTP请求
	body, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("序列化飞书消息失败: %w", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(h.WebhookURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("发送飞书通知失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("飞书通知响应异常: %d", resp.StatusCode)
	}

	return nil
}

// buildFeishuCard 构建飞书卡片消息
func (h *FeishuHook) buildFeishuCard(entry *log.LogEntry) map[string]interface{} {
	appName := h.AppName
	if appName == "" {
		appName = "IGo应用"
	}

	// 根据日志级别选择颜色
	color := "red"
	emoji := "🚨"
	if entry.Level == zapcore.ErrorLevel {
		color = "orange"
		emoji = "⚠️"
	}

	// 构建字段列表
	elements := []interface{}{
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
				"content": fmt.Sprintf("**时间**: %s", entry.Timestamp.Format("2006-01-02 15:04:05")),
				"tag":     "lark_md",
			},
		},
	}

	// 如果有 TraceID，添加到消息中
	if entry.TraceID != "" {
		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"text": map[string]string{
				"content": fmt.Sprintf("**TraceID**: %s", entry.TraceID),
				"tag":     "lark_md",
			},
		})
	}

	// 添加额外的字段信息
	if len(entry.Fields) > 0 {
		fieldsStr := ""
		for k, v := range entry.Fields {
			if k != "traceId" { // traceId 已经单独显示
				fieldsStr += fmt.Sprintf("- %s: %v\n", k, v)
			}
		}
		if fieldsStr != "" {
			elements = append(elements, map[string]interface{}{
				"tag": "div",
				"text": map[string]string{
					"content": fmt.Sprintf("**详细信息**:\n%s", fieldsStr),
					"tag":     "lark_md",
				},
			})
		}
	}

	return map[string]interface{}{
		"msg_type": "interactive",
		"card": map[string]interface{}{
			"header": map[string]interface{}{
				"title": map[string]string{
					"content": fmt.Sprintf("%s [%s] %s", emoji, appName, entry.Level.String()),
					"tag":     "plain_text",
				},
				"template": color,
			},
			"elements": elements,
		},
	}
}

// MockFeishuHook 模拟飞书钩子（用于测试，不实际发送）
type MockFeishuHook struct {
	Messages []string // 保存收到的消息
}

func (h *MockFeishuHook) Levels() []zapcore.Level {
	return []zapcore.Level{
		zapcore.ErrorLevel,
		zapcore.FatalLevel,
	}
}

func (h *MockFeishuHook) Fire(entry *log.LogEntry) error {
	message := fmt.Sprintf("[%s] %s (TraceID: %s)",
		entry.Level.String(),
		entry.Message,
		entry.TraceID,
	)
	h.Messages = append(h.Messages, message)
	fmt.Printf("📱 [模拟飞书通知] %s\n", message)
	return nil
}

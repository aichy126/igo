package log

import (
	"sync"
	"time"

	"go.uber.org/zap/zapcore"
)

// LogHook 日志钩子接口
// 业务层可以实现此接口来处理特定级别的日志，例如发送到飞书、企业微信等
type LogHook interface {
	// Levels 返回该钩子关注的日志级别
	Levels() []zapcore.Level
	// Fire 当有匹配级别的日志时触发
	// 注意：该方法会在独立的goroutine中异步执行，不会阻塞日志记录
	Fire(entry *LogEntry) error
}

// LogEntry 日志条目
type LogEntry struct {
	Level     zapcore.Level          // 日志级别
	Message   string                 // 日志消息
	Fields    map[string]interface{} // 日志字段
	Timestamp time.Time              // 时间戳
	TraceID   string                 // 追踪ID（如果存在）
}

// hookCore 包装原始Core，添加钩子功能
type hookCore struct {
	zapcore.Core
	hooks []LogHook
	mu    sync.RWMutex
}

// newHookCore 创建带钩子的Core
func newHookCore(core zapcore.Core) *hookCore {
	return &hookCore{
		Core:  core,
		hooks: make([]LogHook, 0),
	}
}

// addHook 添加钩子
func (h *hookCore) addHook(hook LogHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.hooks = append(h.hooks, hook)
}

// removeAllHooks 移除所有钩子
func (h *hookCore) removeAllHooks() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.hooks = nil
}

// Write 重写Write方法，在写入日志时触发钩子
func (h *hookCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	// 先调用原始Core的Write
	err := h.Core.Write(entry, fields)

	// 触发钩子（异步执行，不阻塞日志写入）
	h.fireHooks(entry, fields)

	return err
}

// Check 重写Check方法以确保钩子Core的完整性
func (h *hookCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if h.Enabled(entry.Level) {
		return ce.AddCore(entry, h)
	}
	return ce
}

// fireHooks 触发所有匹配级别的钩子
func (h *hookCore) fireHooks(entry zapcore.Entry, fields []zapcore.Field) {
	h.mu.RLock()
	hooks := h.hooks
	h.mu.RUnlock()

	if len(hooks) == 0 {
		return
	}

	// 将zapcore.Field转换为map
	fieldMap := make(map[string]interface{})
	for _, field := range fields {
		// 简化处理，只提取基本类型
		switch field.Type {
		case zapcore.StringType:
			fieldMap[field.Key] = field.String
		case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type:
			fieldMap[field.Key] = field.Integer
		case zapcore.BoolType:
			fieldMap[field.Key] = field.Integer == 1
		default:
			fieldMap[field.Key] = field.Interface
		}
	}

	// 提取traceId
	traceID := ""
	if tid, ok := fieldMap["traceId"]; ok {
		traceID, _ = tid.(string)
	}

	logEntry := &LogEntry{
		Level:     entry.Level,
		Message:   entry.Message,
		Fields:    fieldMap,
		Timestamp: entry.Time,
		TraceID:   traceID,
	}

	// 异步触发钩子，避免阻塞日志记录
	for _, hook := range hooks {
		// 检查是否关注该级别
		if h.shouldFire(hook, entry.Level) {
			go func(hk LogHook, le *LogEntry) {
				defer func() {
					if r := recover(); r != nil {
						// 钩子执行失败不应影响日志记录，静默处理
					}
				}()
				_ = hk.Fire(le)
			}(hook, logEntry)
		}
	}
}

// shouldFire 检查钩子是否应该被触发
func (h *hookCore) shouldFire(hook LogHook, level zapcore.Level) bool {
	levels := hook.Levels()
	for _, l := range levels {
		if l == level {
			return true
		}
	}
	return false
}

var hookCoreInstance *hookCore

// AddHook 添加全局日志钩子
// 使用示例：
//
//	type MyHook struct {}
//	func (h *MyHook) Levels() []zapcore.Level {
//	    return []zapcore.Level{zapcore.ErrorLevel, zapcore.FatalLevel}
//	}
//	func (h *MyHook) Fire(entry *log.LogEntry) error {
//	    // 发送到飞书、企业微信等
//	    return nil
//	}
//	log.AddHook(&MyHook{})
func AddHook(hook LogHook) {
	if hookCoreInstance != nil {
		hookCoreInstance.addHook(hook)
	}
}

// RemoveAllHooks 移除所有钩子
func RemoveAllHooks() {
	if hookCoreInstance != nil {
		hookCoreInstance.removeAllHooks()
	}
}

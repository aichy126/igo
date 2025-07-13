package ictx

import (
	"context"
	"strconv"

	"github.com/aichy126/igo/ilog"
	"github.com/gin-gonic/gin"
)

// Context 增强的上下文接口
type Context interface {
	context.Context

	// Set 设置键值对
	Set(key string, value interface{})

	// Get 获取值
	Get(key string) (interface{}, bool)

	// GetString 获取字符串值
	GetString(key string) string

	// GetInt 获取整数值
	GetInt(key string) int

	// GetAllKey 获取所有键值对
	GetAllKey() map[string]interface{}

	// LogInfo 记录信息日志
	LogInfo(msg string, fields ...log.Field)

	// LogError 记录错误日志
	LogError(msg string, fields ...log.Field)

	// LogWarn 记录警告日志
	LogWarn(msg string, fields ...log.Field)

	// LogDebug 记录调试日志
	LogDebug(msg string, fields ...log.Field)
}

type contextImpl struct {
	context.Context
	data map[string]interface{}
}

// NewContext 创建新的上下文
func NewContext() Context {
	return &contextImpl{
		Context: context.Background(),
		data:    make(map[string]interface{}),
	}
}

// Ginform 从gin.Context创建增强上下文
func Ginform(c *gin.Context) Context {
	// 创建独立的context，避免上游关闭导致runtime断开
	baseCtx := context.Background()

	ctx := &contextImpl{
		Context: baseCtx,
		data:    make(map[string]interface{}),
	}

	// 继承traceId
	if traceId := c.GetString("traceId"); traceId != "" {
		ctx.data["traceId"] = traceId
	}

	// 继承所有HTTP headers（包含userid、token等业务数据）
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			// 如果只有一个值，直接存储字符串；多个值存储切片
			if len(values) == 1 {
				ctx.data[key] = values[0]
			} else {
				ctx.data[key] = values
			}
		}
	}

	// 继承gin context中的所有键值对
	for key, value := range c.Keys {
		ctx.data[key] = value
	}

	return ctx
}

func (c *contextImpl) Set(key string, value interface{}) {
	c.data[key] = value
}

func (c *contextImpl) Get(key string) (interface{}, bool) {
	value, exists := c.data[key]
	return value, exists
}

func (c *contextImpl) GetString(key string) string {
	if value, exists := c.Get(key); exists {
		switch v := value.(type) {
		case string:
			return v
		case []string:
			if len(v) > 0 {
				return v[0]
			}
		}
	}
	return ""
}

func (c *contextImpl) GetInt(key string) int {
	if value, exists := c.Get(key); exists {
		switch v := value.(type) {
		case int:
			return v
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		case []string:
			if len(v) > 0 {
				if i, err := strconv.Atoi(v[0]); err == nil {
					return i
				}
			}
		}
	}
	return 0
}

func (c *contextImpl) GetAllKey() map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range c.data {
		result[k] = v
	}
	return result
}

func (c *contextImpl) LogInfo(msg string, fields ...log.Field) {
	// 添加traceId
	if traceId := c.GetString("traceId"); traceId != "" {
		fields = append(fields, log.String("traceId", traceId))
	}
	log.CtxInfo(msg, fields...)
}

func (c *contextImpl) LogError(msg string, fields ...log.Field) {
	// 添加traceId
	if traceId := c.GetString("traceId"); traceId != "" {
		fields = append(fields, log.String("traceId", traceId))
	}
	log.CtxError(msg, fields...)
}

func (c *contextImpl) LogWarn(msg string, fields ...log.Field) {
	// 添加traceId
	if traceId := c.GetString("traceId"); traceId != "" {
		fields = append(fields, log.String("traceId", traceId))
	}
	log.CtxWarn(msg, fields...)
}

func (c *contextImpl) LogDebug(msg string, fields ...log.Field) {
	// 添加traceId
	if traceId := c.GetString("traceId"); traceId != "" {
		fields = append(fields, log.String("traceId", traceId))
	}
	log.CtxDebug(msg, fields...)
}

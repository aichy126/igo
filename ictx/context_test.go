package ictx

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestNewContext 测试基本的context创建函数
func TestNewContext(t *testing.T) {
	ctx := NewContext()
	assert.NotNil(t, ctx)
}

// TestSetAndGet 测试基本的Set和Get功能
func TestSetAndGet(t *testing.T) {
	ctx := NewContext()

	// 测试字符串
	ctx.Set("key1", "value1")
	val, exists := ctx.Get("key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", val)

	// 测试不存在的key
	_, exists = ctx.Get("nonexistent")
	assert.False(t, exists)
}

// TestGetString 测试字符串类型转换
func TestGetString(t *testing.T) {
	ctx := NewContext()

	// 测试字符串
	ctx.Set("str", "hello")
	assert.Equal(t, "hello", ctx.GetString("str"))

	// 测试字符串数组
	ctx.Set("strarray", []string{"first", "second"})
	assert.Equal(t, "first", ctx.GetString("strarray"))

	// 测试空字符串数组
	ctx.Set("emptyarray", []string{})
	assert.Equal(t, "", ctx.GetString("emptyarray"))

	// 测试不存在的key
	assert.Equal(t, "", ctx.GetString("nonexistent"))
}

// TestGetInt 测试整数类型转换
func TestGetInt(t *testing.T) {
	ctx := NewContext()

	// 测试整数
	ctx.Set("int", 123)
	assert.Equal(t, 123, ctx.GetInt("int"))

	// 测试字符串转换
	ctx.Set("strnum", "567")
	assert.Equal(t, 567, ctx.GetInt("strnum"))

	// 测试字符串数组转换
	ctx.Set("strnumarray", []string{"890", "123"})
	assert.Equal(t, 890, ctx.GetInt("strnumarray"))

	// 测试无效字符串
	ctx.Set("invalidstr", "abc")
	assert.Equal(t, 0, ctx.GetInt("invalidstr"))

	// 测试不存在的key
	assert.Equal(t, 0, ctx.GetInt("nonexistent"))
}

// TestGetAllKey 测试GetAllKey功能
func TestGetAllKey(t *testing.T) {
	ctx := NewContext()

	ctx.Set("key1", "value1")
	ctx.Set("key2", 123)

	allKeys := ctx.GetAllKey()
	assert.Len(t, allKeys, 2)
	assert.Equal(t, "value1", allKeys["key1"])
	assert.Equal(t, 123, allKeys["key2"])
}

// TestGinform 测试Ginform函数
func TestGinform(t *testing.T) {
	// 测试gin.Context参数
	gin.SetMode(gin.TestMode)
	ginCtx, _ := gin.CreateTestContext(nil)

	// 创建HTTP请求
	req, _ := http.NewRequest("GET", "/test", nil)
	ginCtx.Request = req

	// 设置traceId
	ginCtx.Set("traceId", "123")

	ctx := Ginform(ginCtx)
	assert.NotNil(t, ctx)

	// 验证traceId被继承
	traceId := ctx.GetString("traceId")
	assert.Equal(t, "123", traceId)
}

// TestGinformBusinessData 测试Ginform的业务数据继承
func TestGinformBusinessData(t *testing.T) {
	// 创建gin context
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)

	// 模拟HTTP请求，包含业务数据
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Id", "12345")
	req.Header.Set("Authorization", "Bearer abc123token")
	req.Header.Set("X-Custom-Header", "custom-value")
	c.Request = req

	// 设置gin context中的数据
	c.Set("traceId", "trace-123")
	c.Set("session_id", "session-456")

	// 使用Ginform转换
	ctx := Ginform(c)

	// 验证traceId继承
	if traceId := ctx.GetString("traceId"); traceId != "trace-123" {
		t.Errorf("期望traceId='trace-123', 得到='%s'", traceId)
	}

	// 验证HTTP header继承
	if userID := ctx.GetString("User-Id"); userID != "12345" {
		t.Errorf("期望User-Id='12345', 得到='%s'", userID)
	}

	if auth := ctx.GetString("Authorization"); auth != "Bearer abc123token" {
		t.Errorf("期望Authorization='Bearer abc123token', 得到='%s'", auth)
	}

	if custom := ctx.GetString("X-Custom-Header"); custom != "custom-value" {
		t.Errorf("期望X-Custom-Header='custom-value', 得到='%s'", custom)
	}

	// 验证gin context键值对继承
	if sessionID := ctx.GetString("session_id"); sessionID != "session-456" {
		t.Errorf("期望session_id='session-456', 得到='%s'", sessionID)
	}

	t.Logf("✅ 所有业务数据正确继承")

	// 测试context隔离，避免上游关闭导致runtime断开
	if ctx.Err() != nil {
		t.Errorf("context应该是独立的，不应该有错误: %v", ctx.Err())
	}

	t.Logf("✅ Context隔离正常工作")
}

package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAddTraceId(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 测试从请求头获取 traceID
	t.Run("从X-Trace-ID请求头获取", func(t *testing.T) {
		router := gin.New()
		router.Use(AddTraceId())

		var capturedTraceID string
		router.GET("/test", func(c *gin.Context) {
			capturedTraceID = c.GetString("traceId")
			c.Status(200)
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Trace-ID", "existing-trace-123")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, "existing-trace-123", capturedTraceID)
	})

	// 测试生成新的 traceID
	t.Run("没有请求头时生成新traceID", func(t *testing.T) {
		router := gin.New()
		router.Use(AddTraceId())

		var capturedTraceID string
		router.GET("/test", func(c *gin.Context) {
			capturedTraceID = c.GetString("traceId")
			c.Status(200)
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.NotEmpty(t, capturedTraceID)
		assert.Contains(t, capturedTraceID, "-") // UUID格式包含连字符
	})

	// 测试不同格式的请求头
	t.Run("支持不同格式的traceId请求头", func(t *testing.T) {
		testCases := []struct {
			headerName  string
			headerValue string
		}{
			{"X-Trace-Id", "trace-456"},
			{"traceid", "trace-789"},
			{"TraceId", "trace-101"},
		}

		for _, tc := range testCases {
			router := gin.New()
			router.Use(AddTraceId())

			var capturedTraceID string
			router.GET("/test", func(c *gin.Context) {
				capturedTraceID = c.GetString("traceId")
				c.Status(200)
			})

			req, _ := http.NewRequest("GET", "/test", nil)
			req.Header.Set(tc.headerName, tc.headerValue)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.headerValue, capturedTraceID,
				"请求头 %s 应该正确传递 traceID", tc.headerName)
		}
	})
}

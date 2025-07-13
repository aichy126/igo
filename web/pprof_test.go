package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestPprofIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("测试pprof路由注册", func(t *testing.T) {
		router := gin.New()
		WrapPprof(router)

		// 测试pprof首页
		req, _ := http.NewRequest("GET", "/debug/pprof/", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.Contains(t, w.Body.String(), "pprof")
	})

	t.Run("测试heap分析路由", func(t *testing.T) {
		router := gin.New()
		WrapPprof(router)

		req, _ := http.NewRequest("GET", "/debug/pprof/heap", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("测试goroutine分析路由", func(t *testing.T) {
		router := gin.New()
		WrapPprof(router)

		req, _ := http.NewRequest("GET", "/debug/pprof/goroutine", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("测试路由组集成", func(t *testing.T) {
		router := gin.New()
		group := router.Group("/api")
		WrapPprofGroup(group)

		req, _ := http.NewRequest("GET", "/api/debug/pprof/", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})
}

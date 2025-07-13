package web

import (
	"net/http/pprof"
	"strings"

	"github.com/gin-gonic/gin"
)

// WrapPprof 添加pprof路由到gin引擎（简化版本）
func WrapPprof(router *gin.Engine) {
	WrapPprofGroup(&router.RouterGroup)
}

// WrapPprofGroup 添加pprof路由到gin路由组
func WrapPprofGroup(router *gin.RouterGroup) {
	// pprof路由配置
	pprofRoutes := []struct {
		Method  string
		Path    string
		Handler gin.HandlerFunc
	}{
		{"GET", "/debug/pprof/", IndexHandler()},
		{"GET", "/debug/pprof/heap", HeapHandler()},
		{"GET", "/debug/pprof/goroutine", GoroutineHandler()},
		{"GET", "/debug/pprof/block", BlockHandler()},
		{"GET", "/debug/pprof/threadcreate", ThreadCreateHandler()},
		{"GET", "/debug/pprof/cmdline", CmdlineHandler()},
		{"GET", "/debug/pprof/profile", ProfileHandler()},
		{"GET", "/debug/pprof/symbol", SymbolHandler()},
		{"POST", "/debug/pprof/symbol", SymbolHandler()},
		{"GET", "/debug/pprof/trace", TraceHandler()},
		{"GET", "/debug/pprof/mutex", MutexHandler()},
	}

	// 计算路径前缀
	basePath := strings.TrimSuffix(router.BasePath(), "/")
	var prefix string
	switch {
	case basePath == "":
		prefix = ""
	case strings.HasSuffix(basePath, "/debug"):
		prefix = "/debug"
	case strings.HasSuffix(basePath, "/debug/pprof"):
		prefix = "/debug/pprof"
	}

	// 注册路由
	for _, route := range pprofRoutes {
		router.Handle(route.Method, strings.TrimPrefix(route.Path, prefix), route.Handler)
	}
}

// IndexHandler pprof首页处理器
func IndexHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		pprof.Index(c.Writer, c.Request)
	}
}

// HeapHandler 堆内存分析处理器
func HeapHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		pprof.Handler("heap").ServeHTTP(c.Writer, c.Request)
	}
}

// GoroutineHandler goroutine分析处理器
func GoroutineHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		pprof.Handler("goroutine").ServeHTTP(c.Writer, c.Request)
	}
}

// BlockHandler 阻塞分析处理器
func BlockHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		pprof.Handler("block").ServeHTTP(c.Writer, c.Request)
	}
}

// ThreadCreateHandler 线程创建分析处理器
func ThreadCreateHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		pprof.Handler("threadcreate").ServeHTTP(c.Writer, c.Request)
	}
}

// CmdlineHandler 命令行处理器
func CmdlineHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		pprof.Cmdline(c.Writer, c.Request)
	}
}

// ProfileHandler CPU分析处理器
func ProfileHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		pprof.Profile(c.Writer, c.Request)
	}
}

// SymbolHandler 符号表处理器
func SymbolHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		pprof.Symbol(c.Writer, c.Request)
	}
}

// TraceHandler 执行跟踪处理器
func TraceHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		pprof.Trace(c.Writer, c.Request)
	}
}

// MutexHandler 互斥锁分析处理器
func MutexHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		pprof.Handler("mutex").ServeHTTP(c.Writer, c.Request)
	}
}

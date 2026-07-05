package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/aichy126/igo/config"
	"github.com/aichy126/igo/log"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Web
type Web struct {
	Router *gin.Engine
	conf   *config.Config
	server *http.Server
}

// NewWeb
func NewWeb(conf *config.Config) (*Web, error) {
	web := new(Web)
	web.conf = conf
	// gin debug模式
	Debug := conf.GetBool("local.debug")
	if Debug {
		gin.SetMode(gin.DebugMode)
		gin.ForceConsoleColor()
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 用 gin.New() 替代 gin.Default(),自行组装中间件:
	// recovery 始终开启(panic 记录到 zap 日志),避免 gin 默认 logger 与 access 日志双写
	web.Router = gin.New()
	web.Router.Use(AddTraceId())
	web.Router.Use(log.RecoveryWithZap(true))
	if Debug {
		// debug 模式下保留 gin 控制台请求日志,方便本地开发
		web.Router.Use(gin.Logger())
	}

	// pprof 和路由列表有信息泄露风险,仅在 debug 模式或显式配置 local.pprof=true 时注册
	if Debug || conf.GetBool("local.pprof") {
		web.initRouters()
		Wrap(web.Router)
	}

	//Monitor gin logs
	ShowAccess := conf.GetBool("local.logger.access")
	if ShowAccess {
		accesslogger := log.NewAccessLogger(conf)
		web.Router.Use(log.Ginzap(accesslogger, time.RFC3339, true))
	}
	return web, nil
}

const TraceIdHeader = "X-Trace-Id"

// AddTraceId 为每个请求生成/透传 traceId,并写回响应头方便排查问题
func AddTraceId() gin.HandlerFunc {
	return func(g *gin.Context) {
		traceId := g.GetHeader("traceId")
		if traceId == "" {
			traceId = g.GetHeader(TraceIdHeader)
		}
		if traceId == "" {
			traceId = uuid.New().String()
		}
		g.Set("traceId", traceId)
		g.Header(TraceIdHeader, traceId)
		g.Next()
	}
}

func (s *Web) initRouters() {
	s.Router.GET("/debug/http/routers", func(c *gin.Context) {
		routes := s.Router.Routes()
		type routerInfo struct {
			Path    string `json:"path"`
			Handler string `json:"handler"`
			Method  string `json:"method"`
		}
		routerArr := make([]routerInfo, 0)
		for _, r := range routes {
			routerArr = append(routerArr, routerInfo{Path: r.Path, Handler: r.Handler, Method: r.Method})
		}
		c.JSON(http.StatusOK, routerArr)
	})
}

// Run 启动 HTTP 服务(阻塞)
// 正常优雅关闭时返回 nil;启动失败(如端口被占用)返回错误
func (s *Web) Run() error {
	address := s.conf.GetString("local.address")
	if address == "" {
		return fmt.Errorf("缺少配置项 local.address")
	}
	s.server = &http.Server{
		Addr:    address,
		Handler: s.Router,
	}
	fmt.Printf("Gin Address:%s\n", address)
	err := s.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		// 优雅关闭触发的正常返回,不算错误
		return nil
	}
	return err
}

// Shutdown 优雅关闭Web服务器
func (s *Web) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

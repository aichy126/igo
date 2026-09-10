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

const (
	// DefaultReadHeaderTimeout 读完请求头的时限，防慢速连接占着不放
	DefaultReadHeaderTimeout = 20 * time.Second
	// DefaultIdleTimeout 空闲 keep-alive 连接的回收时限。
	// 设了它，关闭时才不会为了等一堆没人用的连接而耗光优雅关闭的时间。
	DefaultIdleTimeout = 60 * time.Second
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
		// 刻意不设 ReadTimeout / WriteTimeout：WriteTimeout 会掐断 SSE 和大文件下载，
		// ReadTimeout 会掐断大文件上传，这两种长连接是业务的正常形态，不该由框架一刀切。
		//
		// IdleTimeout 才是关键：不设的话空闲的 keep-alive 连接会一直挂着，
		// 关闭时 Shutdown 得等它们自己走完，把优雅关闭直接拖成超时。
		ReadHeaderTimeout: DefaultReadHeaderTimeout,
		IdleTimeout:       DefaultIdleTimeout,
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
//
// ctx 到期后会强制断开仍未走完的连接，不再干等。
// 没有这道兜底的话，SSE、websocket、被客户端占着的 keep-alive 连接
// 都会让 Shutdown 一直等到 ctx 超时，而调用方（以及 docker stop）
// 只能陪着一起耗 —— 最后照样是 SIGKILL，白白多花十几秒。
func (s *Web) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	err := s.server.Shutdown(ctx)
	if err != nil {
		// 走到这里说明还有连接赖着不走，直接断掉，让进程能立刻退出
		if cerr := s.server.Close(); cerr != nil {
			log.Error("强制关闭 HTTP 服务失败", log.Any("error", cerr))
		}
	}
	return err
}

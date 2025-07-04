package web

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aichy126/igo/config"
	"github.com/aichy126/igo/log"
	"github.com/aichy126/igo/trace"
	"github.com/gin-gonic/gin"
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
	web.Router = gin.New()

	// 添加中间件
	web.Router.Use(ErrorHandler())
	web.Router.Use(LoggerMiddleware())
	web.Router.Use(SecurityMiddleware())
	web.Router.Use(CORSMiddleware())
	web.Router.Use(AddTraceId())

	web.initRouters()
	Wrap(web.Router)

	//Monitor gin logs
	ShowAccess := conf.GetBool("local.logger.access")
	if ShowAccess {
		AccessFilename := fmt.Sprintf("%s/access.log", conf.GetString("local.logger.dir"))
		Level := conf.GetString("local.logger.level")
		MaxSize := conf.GetInt("local.logger.max_size")
		MaxSizeInt := 1 //Maximum size unit saved per log file: MB
		if MaxSize > 0 {
			MaxSizeInt = MaxSize
		}
		MaxBackups := conf.GetInt("local.logger.max_backups")
		MaxBackupsInt := 5 //The maximum number of days a file can be saved
		if MaxBackups > 0 {
			MaxBackupsInt = MaxBackups
		}
		MaxAge := conf.GetInt("local.logger.max_age")
		MaxAgeInt := 7 //The maximum number of backups saved by the log file
		if MaxAge > 0 {
			MaxAgeInt = MaxAge
		}
		accesslogger := log.InitAccessLogger(AccessFilename, Level, MaxSizeInt, MaxBackupsInt, MaxAgeInt)
		web.Router.Use(log.Ginzap(accesslogger, time.RFC3339, true))
		web.Router.Use(log.RecoveryWithZap(true))
	}
	return web, nil
}

func AddTraceId() gin.HandlerFunc {
	return func(g *gin.Context) {
		// 从请求头提取追踪信息
		ctx, err := trace.ExtractFromHTTPRequest(g.Request)
		if err != nil {
			// 如果提取失败，创建新的追踪上下文
			ctx = g.Request.Context()
		}

		// 开始HTTP请求span
		spanCtx, span := trace.GlobalTracer.StartSpan(ctx, "HTTP "+g.Request.Method+" "+g.Request.URL.Path,
			trace.WithAttributes(map[string]string{
				"http.method":     g.Request.Method,
				"http.url":        g.Request.URL.String(),
				"http.user_agent": g.Request.UserAgent(),
				"http.remote_ip":  g.ClientIP(),
			}),
		)

		// 将追踪上下文设置到gin context
		g.Set("traceContext", spanCtx)
		g.Set("traceSpan", span)

		// 将traceId设置到gin context以保持向后兼容
		traceID := trace.GetTraceID(spanCtx)
		if traceID != "" {
			g.Set("traceId", string(traceID))
		}

		// 处理请求
		g.Next()

		// 结束span
		if span != nil {
			trace.EndSpan(span, nil)
		}
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
		type routerList []routerInfo
		routerArr := make(routerList, 0)
		for _, r := range routes {
			routerArr = append(routerArr, routerInfo{Path: r.Path, Handler: r.Handler, Method: r.Method})
		}
		c.JSON(http.StatusOK, routerArr)
	})

	// 添加健康检查接口
	s.Router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// 添加就绪检查接口
	s.Router.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
			"time":   time.Now().Format(time.RFC3339),
		})
	})
}

func (s *Web) Run() error {
	addr := s.conf.Get("local.address").(string)
	s.server = &http.Server{
		Addr:    addr,
		Handler: s.Router,
	}

	log.Info("Web服务器启动", log.Any("address", addr))
	return s.server.ListenAndServe()
}

// Shutdown 优雅关闭Web服务器
func (s *Web) Shutdown(ctx context.Context) error {
	if s.server != nil {
		log.Info("正在关闭Web服务器...")
		return s.server.Shutdown(ctx)
	}
	return nil
}

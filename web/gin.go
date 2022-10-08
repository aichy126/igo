package web

import (
	"fmt"
	"net/http"
	"time"

	"github.com/aichy126/igo/config"
	"github.com/aichy126/igo/log"
	"github.com/gin-gonic/gin"
)

// Web
type Web struct {
	Router *gin.Engine
	conf   *config.Config
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
	web.Router = gin.Default()
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
}

func (s *Web) Run() {
	s.Router.Run(s.conf.Get("local.address").(string))
}

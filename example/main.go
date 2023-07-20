package main

import (
	"github.com/aichy126/igo"
	"github.com/aichy126/igo/context"
	"github.com/aichy126/igo/log"
	"github.com/aichy126/igo/util"
	"github.com/gin-gonic/gin"
)

func main() {
	igo.App = igo.NewApp("") //初始化各个组件
	debug := util.ConfGetbool("local.debug")
	util.CDump(debug)
	Router(igo.App.Web.Router) //引入 gin路由
	igo.App.Web.Run()
}

func Router(r *gin.Engine) {
	r.GET("ping", Ping)
}

func Ping(c *gin.Context) {
	ctx := context.Ginform(c)
	traceId, has := ctx.Get("traceId")
	log.Info("main-info", log.Any("info", "test"))
	ctx.LogInfo("main-info", log.Any("info", "test"))
	ctx.LogError("main-error", log.Any("error", "test"))
	c.JSON(200, gin.H{
		"message": "Hello World.",
		"traceId": traceId,
		"has":     has,
	})
}

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
	util.Dump(debug)
	Router(igo.App.Web.Router) //引入 gin路由
	igo.App.Web.Run()
}

func Router(r *gin.Engine) {
	r.GET("ping", Ping)
	r.GET("tools", Tools)
}

func Ping(c *gin.Context) {
	ctx := context.Ginform(c)
	traceId, has := ctx.Get("traceId")
	log.Info("main-info", log.Any("info", "test"))
	ctx.LogInfo("main-info", log.Any("info", "test"))
	ctx.LogError("main-error", log.Any("error", "test"))
	util.Dump(traceId, has, ctx)
	c.JSON(200, gin.H{
		"message": "Hello World.",
		"traceId": traceId,
		"has":     has,
	})
}

func Tools(c *gin.Context) {
	num := 123456
	util.Dump(num, util.String(num), util.Int64(util.String(num)))

	c.JSON(200, gin.H{
		"num": util.String(num),
	})
}

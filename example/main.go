package main

import (
	"github.com/aichy126/igo"
	"github.com/aichy126/igo/context"
	"github.com/aichy126/igo/example/dao"
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
	r.GET("db/search", DbSearch)
	r.GET("db/sync", DbSyncSqlite)
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

func DbSearch(c *gin.Context) {
	idStr := c.Query("id")
	id := util.Int64(idStr)
	db := dao.NewTestDbDao()
	ctx := context.Ginform(c)
	info, has, err := db.Info(ctx, id)
	c.JSON(200, gin.H{
		"info": info,
		"has":  has,
		"err":  err,
	})
}

func DbSyncSqlite(c *gin.Context) {
	db := dao.NewTestDbDao()
	ctx := context.Ginform(c)
	err := db.Sync(ctx)
	c.JSON(200, gin.H{
		"err": err,
	})
}

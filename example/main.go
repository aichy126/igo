package main

import (
	"github.com/aichy126/igo"
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
	c.JSON(200, gin.H{"message": "Hello World."})
}

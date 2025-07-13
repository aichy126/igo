package main

import (
	"log"
	"time"

	"github.com/aichy126/igo"
	"github.com/aichy126/igo/ictx"
	ilog "github.com/aichy126/igo/ilog"
	"github.com/gin-gonic/gin"
)

func main() {
	app, err := igo.NewApp("")
	if err != nil {
		log.Fatal("应用初始化失败:", err)
	}
	igo.App = app

	// 设置Consul配置热重载（如果使用Consul配置）
	if igo.App.Conf.GetString("config.address") != "" {
		igo.App.SetConfigHotReloadInterval(60) // 60秒轮询一次
	}

	// 添加配置变更回调（热重载已自动启用）
	igo.App.AddConfigChangeCallback(func() {
		// 配置变更时执行的逻辑
		debug := igo.App.Conf.GetBool("local.debug")
		ilog.Info("配置变更检测", 
			ilog.Bool("debug_mode", debug),
			ilog.String("address", igo.App.Conf.GetString("local.address")))
	})

	// 添加启动钩子
	igo.App.AddStartupHook(func() error {
		ilog.Info("应用启动钩子执行", ilog.Any("time", time.Now()))
		return nil
	})

	// 添加业务关闭钩子（在基础组件关闭之前执行）
	igo.App.AddShutdownHook(func() error {
		ilog.Info("正在关闭业务服务...", ilog.Any("time", time.Now()))
		// 这里可以关闭业务层创建的服务，比如：
		// - 消息队列连接
		// - 定时任务
		// - 自定义服务
		// - 第三方客户端连接
		return nil
	})

	r := igo.App.Web.Router
	registerRoutes(r)

	// 运行应用（自动启动Web服务器并处理优雅关闭）
	if err := igo.App.Run(); err != nil {
		ilog.Error("应用运行失败", ilog.Any("error", err))
	}
}

func registerRoutes(r *gin.Engine) {
	r.GET("/ping", Ping)
	r.GET("/business-span", BusinessSpanDemo)
	r.GET("/trace-demo", TraceDemo)
	r.POST("/reload-config", ReloadConfig) // 手动重载配置接口
}

// 健康检查、优雅关闭、traceId 日志示例
func Ping(c *gin.Context) {
	ctx := ictx.Ginform(c)

	// 简化：直接使用 ctx.LogInfo，自动带 trace 信息
	ctx.LogInfo("ping", ilog.Any("message", "pong"))

	c.JSON(200, gin.H{
		"message": "pong",
		"traceId": c.GetString("traceId"),
	})
}

// 演示简化功能
func BusinessSpanDemo(c *gin.Context) {
	ctx := ictx.Ginform(c)

	ctx.LogInfo("开始用户登录", ilog.String("user_id", "12345"))

	// 模拟业务处理
	time.Sleep(100 * time.Millisecond)

	ctx.LogInfo("用户登录完成")

	c.JSON(200, gin.H{
		"message": "business span demo",
		"traceId": c.GetString("traceId"),
	})
}

// 演示追踪功能（简化版）
func TraceDemo(c *gin.Context) {
	ctx := ictx.Ginform(c)

	ctx.LogInfo("开始处理请求")

	// 模拟一些业务逻辑
	ctx.Set("user_id", "123")
	userID := ctx.GetString("user_id")

	ctx.LogInfo("获取用户信息", ilog.String("user_id", userID))

	c.JSON(200, gin.H{
		"message": "trace demo",
		"user_id": userID,
		"traceId": c.GetString("traceId"),
	})
}

// ReloadConfig 手动重载配置接口
func ReloadConfig(c *gin.Context) {
	ctx := ictx.Ginform(c)
	
	ctx.LogInfo("收到手动重载配置请求")
	
	// 手动重载配置
	if err := igo.App.ReloadConfig(); err != nil {
		ctx.LogError("手动重载配置失败", ilog.String("error", err.Error()))
		c.JSON(500, gin.H{
			"success": false,
			"message": "配置重载失败: " + err.Error(),
			"traceId": c.GetString("traceId"),
		})
		return
	}
	
	ctx.LogInfo("手动重载配置成功")
	c.JSON(200, gin.H{
		"success": true,
		"message": "配置已成功重载",
		"traceId": c.GetString("traceId"),
		"config": gin.H{
			"debug":   igo.App.Conf.GetBool("local.debug"),
			"address": igo.App.Conf.GetString("local.address"),
		},
	})
}

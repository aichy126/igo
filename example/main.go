package main

import (
	"time"

	"github.com/aichy126/igo"
	"github.com/aichy126/igo/context"
	"github.com/aichy126/igo/example/dao"
	"github.com/aichy126/igo/log"
	"github.com/aichy126/igo/trace"
	"github.com/aichy126/igo/util"
	"github.com/gin-gonic/gin"
)

func main() {
	igo.App = igo.NewApp("") // 初始化各个组件
	debug := util.ConfGetbool("local.debug")
	util.Dump(debug)

	// 添加生命周期钩子
	igo.App.AddStartupHook(func() error {
		log.Info("应用启动钩子执行", log.Any("time", time.Now()))
		return nil
	})
	igo.App.AddShutdownHook(func() error {
		log.Info("应用关闭钩子执行", log.Any("time", time.Now()))
		return nil
	})

	r := igo.App.Web.Router
	registerRoutes(r)

	if err := igo.App.RunWithLifecycle(); err != nil {
		log.Error("应用运行失败", log.Any("error", err))
	}
}

func registerRoutes(r *gin.Engine) {
	r.GET("/ping", Ping)
	r.GET("/business-span", BusinessSpanDemo)
	r.GET("/trace-demo", TraceDemo)
	r.GET("/trace-examples", TraceExamples)
	r.GET("db/search", DbSearch)
	r.GET("db/sync", DbSyncSqlite)
}

// 健康检查、优雅关闭、traceId 日志、trace span 示例
func Ping(c *gin.Context) {
	ctx := context.Ginform(c)

	// 简化：直接使用 ctx.LogInfo，自动带 trace 信息
	ctx.LogInfo("ping", log.Any("message", "pong"))

	c.JSON(200, gin.H{
		"message": "pong",
		"traceId": c.GetString("traceId"),
	})
}

// 演示新的业务span功能
func BusinessSpanDemo(c *gin.Context) {
	// 创建带业务span的context
	ctx := context.Ginform(c).StartBusinessSpan("用户登录业务")
	defer ctx.EndBusinessSpan(nil) // 自动结束span

	// 简化：直接使用 ctx.LogInfo，自动带 trace 和 span 信息
	ctx.LogInfo("开始用户登录", log.String("user_id", "12345"))

	// 模拟业务处理
	time.Sleep(100 * time.Millisecond)

	// 记录业务事件
	if span := ctx.GetBusinessSpan(); span != nil {
		trace.AddEvent(span, "用户验证", map[string]string{
			"user_id": "12345",
			"status":  "success",
		})
	}

	ctx.LogInfo("用户登录成功", log.String("user_id", "12345"))

	c.JSON(200, gin.H{
		"message":  "业务span示例",
		"traceId":  c.GetString("traceId"),
		"spanName": "用户登录业务",
	})
}

// traceId 及 span 示例
func TraceDemo(c *gin.Context) {
	ctx := context.Ginform(c)
	traceId := c.GetString("traceId")
	span, _ := c.Get("traceSpan")

	// 简化：直接使用 ctx.LogInfo，自动带 trace 信息
	ctx.LogInfo("trace-demo", log.Any("foo", "bar"))

	// 业务代码可创建子span
	_, childSpan := trace.GlobalTracer.StartSpan(ctx, "业务子操作")
	// ...业务逻辑...
	time.Sleep(50 * time.Millisecond)
	trace.EndSpan(childSpan, nil)

	c.JSON(200, gin.H{
		"traceId": traceId,
		"msg":     "traceId日志和链路追踪示例",
		"spanId":  span,
	})
}

// 演示 ctx.LogInfo 的使用
func TraceExamples(c *gin.Context) {
	ctx := context.Ginform(c)

	// 演示 ctx.LogInfo 自动带 trace 信息
	ctx.LogInfo("开始处理请求", log.String("endpoint", "/trace-examples"))

	// 模拟业务处理
	time.Sleep(50 * time.Millisecond)

	// 记录处理结果
	ctx.LogInfo("请求处理完成", log.String("status", "success"))

	c.JSON(200, gin.H{
		"message": "ctx.LogInfo 示例已执行，请查看日志输出",
		"traceId": c.GetString("traceId"),
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

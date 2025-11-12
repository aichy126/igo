package main

import (
	"net/http"
	"os"

	"github.com/aichy126/igo"
	"github.com/aichy126/igo/context"
	"github.com/aichy126/igo/example/dao"
	"github.com/aichy126/igo/example/hooks"
	"github.com/aichy126/igo/log"
	"github.com/aichy126/igo/util"
	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化应用，现在返回错误而不是panic
	app, err := igo.NewApp("") //初始化各个组件
	if err != nil {
		log.Error("应用初始化失败", log.Any("error", err))
		os.Exit(1)
	}
	igo.App = app

	debug := util.ConfGetbool("local.debug")
	util.Dump(debug)

	// 注册日志钩子（模拟飞书通知）
	mockHook := &hooks.MockFeishuHook{
		Messages: make([]string, 0),
	}
	log.AddHook(mockHook)
	log.Info("日志钩子已注册", log.Any("hook", "MockFeishuHook"))

	// 如果有真实的飞书Webhook URL，可以这样注册：
	// feishuHook := &hooks.FeishuHook{
	//     WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/xxx",
	//     AppName:    "IGo测试应用",
	//     Enabled:    true,
	// }
	// log.AddHook(feishuHook)

	// 添加启动钩子示例
	app.AddStartupHook(func() error {
		log.Info("应用启动完成")

		// 同步数据库表结构
		orderDao := dao.NewOrderDao()
		if err := orderDao.SyncTables(); err != nil {
			log.Error("同步表结构失败", log.Any("error", err))
		}
		return nil
	})

	// 添加配置变更回调示例
	app.AddConfigChangeCallback(func() {
		log.Info("配置已更新")
	})

	// 添加关闭钩子示例
	app.AddShutdownHook(func() error {
		log.Info("应用准备关闭，执行清理工作...")
		return nil
	})

	Router(app.Web.Router) //引入 gin路由

	// 使用新的 RunWithGracefulShutdown 方法，自动处理优雅关闭
	if err := app.RunWithGracefulShutdown(); err != nil {
		log.Error("应用运行失败", log.Any("error", err))
		os.Exit(1)
	}
}

func Router(r *gin.Engine) {
	// 原有路由
	r.GET("ping", Ping)
	r.GET("db/search", DbSearch)
	r.GET("db/sync", DbSyncSqlite)

	// 新功能测试路由
	r.POST("order/create", CreateOrder)           // 跨表事务：创建订单
	r.POST("db/batch-sync", BatchSyncData)        // 跨表事务：批量同步
	r.GET("test/log-hook", TestLogHook)           // 测试日志钩子
	r.POST("config/reload", ReloadConfig)         // 配置热重载
	r.GET("health", HealthCheck)                  // 健康检查
	r.GET("middleware/test", MiddlewareTest)      // 测试中间件
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

// CreateOrder 创建订单（跨表事务示例）
func CreateOrder(c *gin.Context) {
	ctx := context.Ginform(c)

	// 解析请求参数
	type CreateOrderReq struct {
		UserID string `json:"user_id" binding:"required"`
		Items  []struct {
			Product  string  `json:"product" binding:"required"`
			Quantity int     `json:"quantity" binding:"required,min=1"`
			Price    float64 `json:"price" binding:"required,min=0"`
		} `json:"items" binding:"required,min=1"`
	}

	var req CreateOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误",
			"error":   err.Error(),
		})
		return
	}

	// 转换为订单项
	var items []dao.OrderItem
	for _, item := range req.Items {
		items = append(items, dao.OrderItem{
			Product:  item.Product,
			Quantity: item.Quantity,
			Price:    item.Price,
		})
	}

	// 创建订单（使用跨表事务）
	orderDao := dao.NewOrderDao()
	order, err := orderDao.CreateOrderWithItems(ctx, req.UserID, items)
	if err != nil {
		ctx.LogError("创建订单失败", log.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建订单失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "订单创建成功",
		"data":    order,
	})
}

// BatchSyncData 批量同步数据（跨表事务示例）
func BatchSyncData(c *gin.Context) {
	ctx := context.Ginform(c)
	ctx.LogInfo("收到批量同步请求")

	orderDao := dao.NewOrderDao()
	err := orderDao.BatchSync(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "同步失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "同步成功",
	})
}

// TestLogHook 测试日志钩子
func TestLogHook(c *gin.Context) {
	ctx := context.Ginform(c)

	// 触发不同级别的日志
	ctx.LogInfo("这是一条Info日志，不会触发钩子")
	log.Warn("这是一条Warn日志，不会触发钩子")

	// 这些日志会触发钩子（如果注册了Error级别的钩子）
	ctx.LogError("测试错误日志 - 这应该触发钩子", log.Any("test_field", "test_value"))

	// 模拟一个业务错误
	log.Error("模拟业务错误：数据库连接失败",
		log.Any("error", "connection timeout"),
		log.Any("database", "test"),
		log.Any("retry_count", 3),
	)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "日志钩子测试完成，请查看飞书/企微是否收到通知",
	})
}

// ReloadConfig 手动重载配置
func ReloadConfig(c *gin.Context) {
	ctx := context.Ginform(c)
	ctx.LogInfo("手动重载配置")

	if err := igo.App.ReloadConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "配置重载失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "配置重载成功",
	})
}

// HealthCheck 健康检查
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"message": "应用运行正常",
	})
}

// MiddlewareTest 测试中间件
func MiddlewareTest(c *gin.Context) {
	ctx := context.Ginform(c)

	// 测试中间件是否正常工作
	traceId, _ := ctx.Get("traceId")

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "中间件测试",
		"data": gin.H{
			"trace_id": traceId,
			"headers":  c.Request.Header,
		},
	})
}

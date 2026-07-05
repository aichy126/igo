package igo

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aichy126/igo/cache"
	"github.com/aichy126/igo/config"
	"github.com/aichy126/igo/db"
	"github.com/aichy126/igo/lifecycle"
	"github.com/aichy126/igo/log"
	"github.com/aichy126/igo/web"
	"github.com/gin-gonic/gin"
)

type Application struct {
	Conf  *config.Config
	Web   *web.Web
	DB    *db.DB
	Cache *cache.Cache
	// 生命周期管理器
	lifecycle *lifecycle.LifecycleManager
}

var App *Application

// NewApp 创建应用实例
// 初始化顺序:config → log → db → cache → web
// 配置了的组件初始化失败会返回错误(fail-fast);db/redis 未配置时跳过,不报错
func NewApp(ConfigPath string) (*Application, error) {
	a := new(Application)

	//config
	conf, err := config.NewConfig(ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("配置文件加载失败: %w", err)
	}

	// 验证配置
	if err := conf.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	a.Conf = conf

	// 根据配置源类型自动启动热重载监听
	if a.Conf.IsHotReloadEnabled() {
		a.Conf.WatchConfig()
	}

	//log(最先初始化,后续组件初始化失败时日志可用)
	_, err = log.NewLog(conf)
	if err != nil {
		return nil, fmt.Errorf("日志系统初始化失败: %w", err)
	}

	// 配置热重载时同步日志级别:改 local.logger.level 即时生效,无需重启
	a.Conf.AddChangeCallback(func() {
		if lvl := a.Conf.GetString("local.logger.level"); lvl != "" {
			if err := log.SetLevel(lvl); err != nil {
				log.Warn("日志级别热更新失败", log.Any("error", err))
			}
		}
	})

	//db(配置了就必须初始化成功,未配置返回空实例)
	dbInstance, err := db.NewDb(conf)
	if err != nil {
		return nil, fmt.Errorf("数据库初始化失败: %w", err)
	}
	a.DB = dbInstance

	//cache(配置了就必须初始化成功,未配置返回空实例)
	cacheInstance, err := cache.NewCache(conf)
	if err != nil {
		return nil, fmt.Errorf("缓存初始化失败: %w", err)
	}
	a.Cache = cacheInstance

	//web
	webInstance, err := web.NewWeb(conf)
	if err != nil {
		return nil, fmt.Errorf("Web服务初始化失败: %w", err)
	}
	a.Web = webInstance

	// 初始化生命周期管理器
	a.lifecycle = lifecycle.NewLifecycleManager()

	// 自动注册所有组件的优雅关闭（按依赖关系反向顺序执行:后注册的先关闭）
	// 关闭顺序:Web(停止接收新请求) → Cache → DB
	a.lifecycle.AddShutdownHook(func() error {
		return a.DB.Close()
	})
	a.lifecycle.AddShutdownHook(func() error {
		return a.Cache.Close()
	})
	a.lifecycle.AddShutdownHook(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), lifecycle.DefaultShutdownTimeout)
		defer cancel()
		return a.Web.Shutdown(ctx)
	})

	// 设置全局应用实例
	App = a

	return a, nil
}

// Run 运行应用:启动Web服务器,等待退出信号后优雅关闭
// Web 启动失败(如端口被占用)会立即返回错误,不会挂起等待信号
func (a *Application) Run() error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- a.Web.Run()
	}()

	// 运行生命周期管理器(等待信号或运行错误,并处理优雅关闭)
	return a.lifecycle.Run(errCh)
}

// RunWithGracefulShutdown 直接运行应用并处理优雅关闭
// Deprecated: 与 Run 等价,请直接使用 Run
func (a *Application) RunWithGracefulShutdown() error {
	return a.Run()
}

// Shutdown 主动关闭应用(带超时)
func (a *Application) Shutdown() error {
	return a.lifecycle.GracefulShutdown(10 * time.Second)
}

// GetShutdownContext 获取关闭上下文,应用开始关闭时会被 cancel
// 业务方可以监听它来停止自己的后台 goroutine
func (a *Application) GetShutdownContext() context.Context {
	return a.lifecycle.GetShutdownContext()
}

// AddShutdownHook 添加关闭钩子
func (a *Application) AddShutdownHook(hook func() error) {
	if a.lifecycle != nil {
		a.lifecycle.AddShutdownHook(hook)
	}
}

// AddStartupHook 添加启动钩子
func (a *Application) AddStartupHook(hook func() error) {
	if a.lifecycle != nil {
		a.lifecycle.AddStartupHook(hook)
	}
}

// AddConfigChangeCallback 添加配置变更回调
func (a *Application) AddConfigChangeCallback(callback func()) {
	if a.Conf != nil {
		a.Conf.AddChangeCallback(callback)
	}
}

// SetConfigHotReloadInterval 设置配置热重载轮询间隔
// intervalSeconds: 轮询间隔秒数，-1或0表示禁用热重载，>0表示启用并设置间隔
// 注意：仅对Consul配置有效，文件配置自动启用热重载
func (a *Application) SetConfigHotReloadInterval(intervalSeconds int) *Application {
	if a.Conf != nil {
		a.Conf.SetHotReloadInterval(intervalSeconds)
		// 如果设置了有效间隔且之前没有启动监听，现在启动
		if intervalSeconds > 0 && a.Conf.IsHotReloadEnabled() {
			a.Conf.WatchConfig()
		}
	}
	return a
}

// ReloadConfig 手动重新加载配置
func (a *Application) ReloadConfig() error {
	if a.Conf != nil {
		return a.Conf.ReloadConfig()
	}
	return fmt.Errorf("配置实例不存在")
}

// EnableHealthCheck 注册 GET /health 健康检查路由(可选,一行开启)
// 返回应用状态以及所有已配置 db/redis 的连通性;任一组件异常时返回 503
func (a *Application) EnableHealthCheck() *Application {
	a.Web.Router.GET("/health", func(c *gin.Context) {
		healthy := true
		components := gin.H{}

		if a.DB != nil && a.DB.DBResourceManager != nil {
			for name, err := range a.DB.PingAll() {
				key := "db:" + name
				if err != nil {
					healthy = false
					components[key] = err.Error()
				} else {
					components[key] = "ok"
				}
			}
		}

		if a.Cache != nil && a.Cache.RedisManager != nil {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()
			for name, err := range a.Cache.PingAll(ctx) {
				key := "redis:" + name
				if err != nil {
					healthy = false
					components[key] = err.Error()
				} else {
					components[key] = "ok"
				}
			}
		}

		status := http.StatusOK
		statusText := "healthy"
		if !healthy {
			status = http.StatusServiceUnavailable
			statusText = "unhealthy"
		}
		c.JSON(status, gin.H{
			"status":     statusText,
			"components": components,
		})
	})
	return a
}

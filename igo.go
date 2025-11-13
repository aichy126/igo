package igo

import (
	"context"
	"fmt"
	"time"

	"github.com/aichy126/igo/cache"
	"github.com/aichy126/igo/config"
	"github.com/aichy126/igo/db"
	"github.com/aichy126/igo/lifecycle"
	"github.com/aichy126/igo/log"
	"github.com/aichy126/igo/web"
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
		go a.Conf.WatchConfig()
	}

	//web
	web, err := web.NewWeb(conf)
	if err != nil {
		return nil, fmt.Errorf("Web服务初始化失败: %w", err)
	}
	a.Web = web

	//log
	_, err = log.NewLog(conf)
	if err != nil {
		return nil, fmt.Errorf("日志系统初始化失败: %w", err)
	}

	//db (可选组件，失败时静默处理，不提醒)
	dbInstance, err := db.NewDb(conf)
	if err != nil {
		// 静默处理：创建一个空的db实例，避免nil指针
		a.DB = &db.DB{}
	} else {
		a.DB = dbInstance
	}

	//cache (可选组件，失败时静默处理，不提醒)
	cacheInstance, err := cache.NewCache(conf)
	if err != nil {
		// 静默处理：创建一个空的cache实例，避免nil指针
		a.Cache = &cache.Cache{}
	} else {
		a.Cache = cacheInstance
	}

	// 初始化生命周期管理器（传入nil，简化接口）
	a.lifecycle = lifecycle.NewLifecycleManager(nil)

	// 自动注册所有组件的优雅关闭（按依赖关系反向顺序）
	// 1. 首先关闭Web服务器（停止接收新请求）
	a.lifecycle.AddShutdownHook(func() error {
		log.Info("正在关闭Web服务器...")
		return a.Web.Shutdown(context.Background())
	})

	// 2. 然后关闭缓存连接（如果存在）
	if a.Cache != nil && a.Cache.RedisManager != nil {
		a.lifecycle.AddShutdownHook(func() error {
			log.Info("正在关闭缓存连接...")
			return a.Cache.Close()
		})
	}

	// 3. 最后关闭数据库连接（如果存在）
	if a.DB != nil && a.DB.DBResourceManager != nil {
		a.lifecycle.AddShutdownHook(func() error {
			log.Info("正在关闭数据库连接...")
			return a.DB.Close()
		})
	}

	// 设置全局应用实例
	App = a

	return a, nil
}

// Run 运行应用（启动Web服务器并等待关闭信号）
func (a *Application) Run() error {
	// 在goroutine中启动Web服务器
	go func() {
		if err := a.Web.Run(); err != nil {
			log.Error("Web服务器运行失败", log.Any("error", err))
		}
	}()

	// 运行生命周期管理器（等待信号并处理优雅关闭）
	return a.lifecycle.Run()
}

// RunWithGracefulShutdown 直接运行应用并处理优雅关闭（简化用法）
func (a *Application) RunWithGracefulShutdown() error {
	return a.Run()
}

// Shutdown 关闭应用
func (a *Application) Shutdown() error {
	return a.lifecycle.GracefulShutdown(10 * time.Second)
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
			go a.Conf.WatchConfig()
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

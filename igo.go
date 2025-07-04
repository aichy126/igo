package igo

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
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
	// 添加关闭相关的字段
	shutdownCtx    context.Context
	shutdownCancel context.CancelFunc
	// 添加生命周期管理器
	lifecycle *lifecycle.LifecycleManager
}

var App *Application

func NewApp(ConfigPath string) *Application {
	a := new(Application)
	// 创建关闭上下文
	a.shutdownCtx, a.shutdownCancel = context.WithCancel(context.Background())

	//config
	conf, err := config.NewConfig(ConfigPath)
	if err != nil {
		panic(err)
	}

	// 验证配置
	if err := conf.Validate(); err != nil {
		panic(fmt.Sprintf("配置验证失败: %v", err))
	}

	a.Conf = conf

	//web
	web, err := web.NewWeb(conf)
	if err != nil {
		panic(err)
	}
	a.Web = web

	//log
	_, err = log.NewLog(conf)
	if err != nil {
		panic(err)
	}

	//db
	db, err := db.NewDb(conf)
	if err != nil {
		panic(err)
	}
	a.DB = db

	//cahce
	cache, err := cache.NewCache(conf)
	if err != nil {
		panic(err)
	}
	a.Cache = cache

	// 初始化生命周期管理器
	a.lifecycle = lifecycle.NewLifecycleManager(a)

	return a
}

// GracefulShutdown 优雅关闭应用
func (a *Application) GracefulShutdown(timeout time.Duration) error {
	log.Info("开始优雅关闭应用...")

	// 设置关闭超时
	ctx, cancel := context.WithTimeout(a.shutdownCtx, timeout)
	defer cancel()

	// 关闭Web服务器
	if a.Web != nil {
		log.Info("正在关闭Web服务器...")
		if err := a.Web.Shutdown(ctx); err != nil {
			log.Error("关闭Web服务器失败", log.Any("error", err))
		}
	}

	// 关闭数据库连接
	if a.DB != nil {
		log.Info("正在关闭数据库连接...")
		if err := a.DB.Close(); err != nil {
			log.Error("关闭数据库连接失败", log.Any("error", err))
		}
	}

	// 关闭缓存连接
	if a.Cache != nil {
		log.Info("正在关闭缓存连接...")
		if err := a.Cache.Close(); err != nil {
			log.Error("关闭缓存连接失败", log.Any("error", err))
		}
	}

	log.Info("应用已优雅关闭")
	return nil
}

// RunWithGracefulShutdown 运行应用并处理优雅关闭
func (a *Application) RunWithGracefulShutdown() {
	// 启动Web服务器
	go func() {
		if err := a.Web.Run(); err != nil {
			log.Error("Web服务器启动失败", log.Any("error", err))
		}
	}()

	// 等待关闭信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("收到关闭信号，开始优雅关闭...")

	// 触发关闭上下文
	a.shutdownCancel()

	// 执行优雅关闭
	if err := a.GracefulShutdown(30 * time.Second); err != nil {
		log.Error("优雅关闭失败", log.Any("error", err))
		os.Exit(1)
	}
}

// GetShutdownContext 获取关闭上下文
func (a *Application) GetShutdownContext() context.Context {
	return a.shutdownCtx
}

// GetLifecycleManager 获取生命周期管理器
func (a *Application) GetLifecycleManager() *lifecycle.LifecycleManager {
	return a.lifecycle
}

// GetWeb 获取Web服务器（实现AppInterface）
func (a *Application) GetWeb() interface {
	Run() error
	Shutdown(ctx context.Context) error
} {
	return a.Web
}

// GetDB 获取数据库（实现AppInterface）
func (a *Application) GetDB() interface {
	Close() error
} {
	return a.DB
}

// GetCache 获取缓存（实现AppInterface）
func (a *Application) GetCache() interface {
	Close() error
} {
	return a.Cache
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

// RunWithLifecycle 使用生命周期管理器运行应用
func (a *Application) RunWithLifecycle() error {
	return a.lifecycle.Run()
}

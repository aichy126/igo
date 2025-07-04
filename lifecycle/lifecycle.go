package lifecycle

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/aichy126/igo/log"
)

// AppInterface 应用接口，避免循环导入
type AppInterface interface {
	GetWeb() interface {
		Run() error
		Shutdown(ctx context.Context) error
	}
	GetDB() interface {
		Close() error
	}
	GetCache() interface {
		Close() error
	}
}

// LifecycleManager 应用生命周期管理器
type LifecycleManager struct {
	app           AppInterface
	shutdownHooks []func() error
	startupHooks  []func() error
	mu            sync.RWMutex
}

// NewLifecycleManager 创建生命周期管理器
func NewLifecycleManager(app AppInterface) *LifecycleManager {
	return &LifecycleManager{
		app:           app,
		shutdownHooks: make([]func() error, 0),
		startupHooks:  make([]func() error, 0),
	}
}

// AddShutdownHook 添加关闭钩子
func (lm *LifecycleManager) AddShutdownHook(hook func() error) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.shutdownHooks = append(lm.shutdownHooks, hook)
}

// AddStartupHook 添加启动钩子
func (lm *LifecycleManager) AddStartupHook(hook func() error) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.startupHooks = append(lm.startupHooks, hook)
}

// Run 运行应用
func (lm *LifecycleManager) Run() error {
	// 执行启动钩子
	if err := lm.executeStartupHooks(); err != nil {
		return fmt.Errorf("启动钩子执行失败: %w", err)
	}

	// 启动Web服务器
	go func() {
		if err := lm.app.GetWeb().Run(); err != nil {
			log.Error("Web服务器启动失败", log.Any("error", err))
		}
	}()

	// 等待关闭信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("收到关闭信号，开始优雅关闭...")

	// 执行优雅关闭
	if err := lm.GracefulShutdown(30 * time.Second); err != nil {
		log.Error("优雅关闭失败", log.Any("error", err))
		return err
	}

	return nil
}

// GracefulShutdown 优雅关闭
func (lm *LifecycleManager) GracefulShutdown(timeout time.Duration) error {
	log.Info("开始优雅关闭应用...")

	// 设置关闭超时
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 执行关闭钩子
	if err := lm.executeShutdownHooks(); err != nil {
		log.Error("执行关闭钩子失败", log.Any("error", err))
	}

	// 关闭Web服务器
	if lm.app.GetWeb() != nil {
		log.Info("正在关闭Web服务器...")
		if err := lm.app.GetWeb().Shutdown(ctx); err != nil {
			log.Error("关闭Web服务器失败", log.Any("error", err))
		}
	}

	// 关闭数据库连接
	if lm.app.GetDB() != nil {
		log.Info("正在关闭数据库连接...")
		if err := lm.app.GetDB().Close(); err != nil {
			log.Error("关闭数据库连接失败", log.Any("error", err))
		}
	}

	// 关闭缓存连接
	if lm.app.GetCache() != nil {
		log.Info("正在关闭缓存连接...")
		if err := lm.app.GetCache().Close(); err != nil {
			log.Error("关闭缓存连接失败", log.Any("error", err))
		}
	}

	log.Info("应用已优雅关闭")
	return nil
}

// executeStartupHooks 执行启动钩子
func (lm *LifecycleManager) executeStartupHooks() error {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	for i, hook := range lm.startupHooks {
		log.Info("执行启动钩子", log.Any("index", i))
		if err := hook(); err != nil {
			return fmt.Errorf("启动钩子 %d 执行失败: %w", i, err)
		}
	}
	return nil
}

// executeShutdownHooks 执行关闭钩子
func (lm *LifecycleManager) executeShutdownHooks() error {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	// 反向执行关闭钩子
	for i := len(lm.shutdownHooks) - 1; i >= 0; i-- {
		hook := lm.shutdownHooks[i]
		log.Info("执行关闭钩子", log.Any("index", i))
		if err := hook(); err != nil {
			log.Error("关闭钩子执行失败", log.Any("index", i), log.Any("error", err))
		}
	}
	return nil
}

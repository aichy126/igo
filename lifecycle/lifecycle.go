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

// LifecycleManager 应用生命周期管理器
type LifecycleManager struct {
	shutdownHooks  []func() error
	startupHooks   []func() error
	mu             sync.RWMutex
	shutdownCtx    context.Context
	shutdownCancel context.CancelFunc
}

// NewLifecycleManager 创建生命周期管理器
func NewLifecycleManager(_ interface{}) *LifecycleManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &LifecycleManager{
		shutdownHooks:  make([]func() error, 0),
		startupHooks:   make([]func() error, 0),
		shutdownCtx:    ctx,
		shutdownCancel: cancel,
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

// Run 运行应用（简化版本，只处理钩子和信号）
func (lm *LifecycleManager) Run() error {
	// 执行启动钩子
	if err := lm.executeStartupHooks(); err != nil {
		return fmt.Errorf("启动钩子执行失败: %w", err)
	}

	// 等待关闭信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("收到关闭信号，开始执行关闭钩子...")

	// 执行关闭钩子
	if err := lm.executeShutdownHooks(); err != nil {
		log.Error("关闭钩子执行失败", log.Any("error", err))
		return err
	}

	return nil
}

// GracefulShutdown 优雅关闭（简化版本）
func (lm *LifecycleManager) GracefulShutdown(timeout time.Duration) error {
	log.Info("开始优雅关闭应用...")

	// 触发关闭上下文
	lm.shutdownCancel()

	// 执行关闭钩子
	if err := lm.executeShutdownHooks(); err != nil {
		log.Error("执行关闭钩子失败", log.Any("error", err))
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

// GetShutdownContext 获取关闭上下文
func (lm *LifecycleManager) GetShutdownContext() context.Context {
	return lm.shutdownCtx
}

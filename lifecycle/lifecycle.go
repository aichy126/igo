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

// DefaultShutdownTimeout 默认优雅关闭超时时间
const DefaultShutdownTimeout = 10 * time.Second

// LifecycleManager 应用生命周期管理器
type LifecycleManager struct {
	shutdownHooks   []func() error
	startupHooks    []func() error
	mu              sync.RWMutex
	shutdownCtx     context.Context
	shutdownCancel  context.CancelFunc
	shutdownOnce    sync.Once
	shutdownTimeout time.Duration
}

// NewLifecycleManager 创建生命周期管理器
func NewLifecycleManager() *LifecycleManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &LifecycleManager{
		shutdownHooks:   make([]func() error, 0),
		startupHooks:    make([]func() error, 0),
		shutdownCtx:     ctx,
		shutdownCancel:  cancel,
		shutdownTimeout: DefaultShutdownTimeout,
	}
}

// SetShutdownTimeout 设置优雅关闭超时时间
func (lm *LifecycleManager) SetShutdownTimeout(d time.Duration) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	if d > 0 {
		lm.shutdownTimeout = d
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

// Run 运行生命周期:执行启动钩子,等待退出信号或运行时错误,然后优雅关闭
// errCh 用于接收组件的运行时错误(如 web 服务器启动失败),收到非 nil 错误会触发关闭并返回该错误
func (lm *LifecycleManager) Run(errCh <-chan error) error {
	// 执行启动钩子
	if err := lm.executeStartupHooks(); err != nil {
		return fmt.Errorf("启动钩子执行失败: %w", err)
	}

	// 等待关闭信号或运行时错误
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	var runErr error
	select {
	case sig := <-quit:
		fmt.Printf("\n收到信号 %s,开始关闭应用...\n", sig)
	case err := <-errCh:
		if err == nil {
			// web 服务器正常退出(如外部调用 Shutdown),照常执行关闭流程
			fmt.Printf("\n服务已停止,开始关闭应用...\n")
		} else {
			runErr = err
			log.Error("服务运行失败,开始关闭应用", log.Any("error", err))
		}
	}

	// 执行关闭钩子(带超时)
	if err := lm.shutdown(); err != nil && runErr == nil {
		runErr = err
	}
	return runErr
}

// GracefulShutdown 优雅关闭(带超时)
func (lm *LifecycleManager) GracefulShutdown(timeout time.Duration) error {
	if timeout > 0 {
		lm.SetShutdownTimeout(timeout)
	}
	return lm.shutdown()
}

// shutdown 执行关闭流程,shutdownOnce 保证只执行一次
func (lm *LifecycleManager) shutdown() error {
	var err error
	lm.shutdownOnce.Do(func() {
		// 触发关闭上下文,通知业务方(通过 GetShutdownContext 获取)
		lm.shutdownCancel()

		lm.mu.RLock()
		timeout := lm.shutdownTimeout
		lm.mu.RUnlock()

		done := make(chan error, 1)
		go func() {
			done <- lm.executeShutdownHooks()
		}()

		select {
		case err = <-done:
			log.Info("应用已优雅关闭")
		case <-time.After(timeout):
			err = fmt.Errorf("优雅关闭超时(%s),强制退出", timeout)
			log.Error("优雅关闭超时", log.Any("timeout", timeout.String()))
		}
	})
	return err
}

// executeStartupHooks 执行启动钩子
func (lm *LifecycleManager) executeStartupHooks() error {
	lm.mu.RLock()
	hooks := make([]func() error, len(lm.startupHooks))
	copy(hooks, lm.startupHooks)
	lm.mu.RUnlock()

	for i, hook := range hooks {
		if err := hook(); err != nil {
			return fmt.Errorf("启动钩子 %d 执行失败: %w", i, err)
		}
	}
	return nil
}

// executeShutdownHooks 执行关闭钩子(按注册顺序反向执行)
func (lm *LifecycleManager) executeShutdownHooks() error {
	lm.mu.RLock()
	hooks := make([]func() error, len(lm.shutdownHooks))
	copy(hooks, lm.shutdownHooks)
	lm.mu.RUnlock()

	var firstErr error
	for i := len(hooks) - 1; i >= 0; i-- {
		if err := hooks[i](); err != nil {
			log.Error("关闭钩子执行失败", log.Any("index", i), log.Any("error", err))
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// GetShutdownContext 获取关闭上下文
// 应用开始关闭时该 context 会被 cancel,业务方可以监听它来停止后台任务
func (lm *LifecycleManager) GetShutdownContext() context.Context {
	return lm.shutdownCtx
}

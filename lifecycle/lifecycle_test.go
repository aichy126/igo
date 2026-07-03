package lifecycle

import (
	"errors"
	"testing"
	"time"
)

// TestRunReturnsOnError 验证组件运行错误(如端口占用)会让 Run 立即返回,而不是挂起等信号
func TestRunReturnsOnError(t *testing.T) {
	lm := NewLifecycleManager()

	var closed bool
	lm.AddShutdownHook(func() error {
		closed = true
		return nil
	})

	errCh := make(chan error, 1)
	errCh <- errors.New("listen tcp: address already in use")

	done := make(chan error, 1)
	go func() { done <- lm.Run(errCh) }()

	select {
	case err := <-done:
		if err == nil {
			t.Error("Run 应返回组件错误")
		}
		if !closed {
			t.Error("出错退出时也应执行关闭钩子")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Run 未在组件出错后返回(挂起等信号)")
	}
}

// TestShutdownHooksReverseOrder 验证关闭钩子按注册反序执行
func TestShutdownHooksReverseOrder(t *testing.T) {
	lm := NewLifecycleManager()
	var order []int
	for i := 0; i < 3; i++ {
		i := i
		lm.AddShutdownHook(func() error {
			order = append(order, i)
			return nil
		})
	}
	if err := lm.GracefulShutdown(time.Second); err != nil {
		t.Fatalf("GracefulShutdown error: %v", err)
	}
	if len(order) != 3 || order[0] != 2 || order[2] != 0 {
		t.Errorf("关闭顺序错误: %v", order)
	}
}

// TestShutdownTimeout 验证优雅关闭超时生效
func TestShutdownTimeout(t *testing.T) {
	lm := NewLifecycleManager()
	lm.AddShutdownHook(func() error {
		time.Sleep(5 * time.Second)
		return nil
	})
	start := time.Now()
	err := lm.GracefulShutdown(200 * time.Millisecond)
	if err == nil {
		t.Error("超时应返回错误")
	}
	if time.Since(start) > 2*time.Second {
		t.Error("超时未生效")
	}
}

// TestShutdownOnlyOnce 验证关闭流程只执行一次
func TestShutdownOnlyOnce(t *testing.T) {
	lm := NewLifecycleManager()
	count := 0
	lm.AddShutdownHook(func() error {
		count++
		return nil
	})
	_ = lm.GracefulShutdown(time.Second)
	_ = lm.GracefulShutdown(time.Second)
	if count != 1 {
		t.Errorf("关闭钩子执行了 %d 次, 应为 1 次", count)
	}
}

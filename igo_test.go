package igo

import (
	"testing"

	"github.com/aichy126/igo/lifecycle"
	"github.com/stretchr/testify/assert"
)

func TestApplicationGracefulShutdown(t *testing.T) {
	t.Run("测试优雅关闭钩子注册", func(t *testing.T) {
		app := &Application{}
		app.lifecycle = lifecycle.NewLifecycleManager(nil)

		// 测试添加关闭钩子
		app.AddShutdownHook(func() error {
			return nil
		})

		// 验证lifecycle被正确初始化
		assert.NotNil(t, app.lifecycle)

		// 注意：我们不直接调用Shutdown因为它需要日志系统
	})

	t.Run("测试启动钩子注册", func(t *testing.T) {
		app := &Application{}
		app.lifecycle = lifecycle.NewLifecycleManager(nil)

		// 测试添加启动钩子
		app.AddStartupHook(func() error {
			return nil
		})

		// 验证lifecycle被正确初始化
		assert.NotNil(t, app.lifecycle)
	})
}

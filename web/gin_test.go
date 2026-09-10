package web

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

// newTestWeb 起一个真实监听的 Web，handler 由调用方给
func newTestWeb(t *testing.T, h http.Handler) (*Web, string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("监听失败: %v", err)
	}
	w := &Web{server: &http.Server{Handler: h}}
	go func() { _ = w.server.Serve(ln) }()
	t.Cleanup(func() { _ = w.server.Close() })
	return w, ln.Addr().String()
}

// 关闭超时后必须真的把赖着不走的连接断掉。
//
// 标准库的 Shutdown 在 ctx 到期时只是「返回错误」，那些连接和它们的 goroutine
// 依然挂着、客户端也依然傻等 —— 进程于是迟迟退不干净，docker stop 陪着一起耗，
// 最后照样吃 SIGKILL。所以超时之后要补一刀 Close()。
//
// 这个测试盯的就是那一刀：去掉 Close() 兜底它就会失败。
func TestShutdownTimeoutClosesLingeringConnections(t *testing.T) {
	release := make(chan struct{})
	defer close(release)

	started := make(chan struct{})
	w, addr := newTestWeb(t, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
		rw.(http.Flusher).Flush() // 先把响应头发出去，让客户端进入读 body 的状态
		close(started)
		<-release // 模拟迟迟不结束的请求：SSE、长轮询都是这个形态
	}))

	clientDone := make(chan error, 1)
	go func() {
		resp, err := http.Get("http://" + addr + "/")
		if err != nil {
			clientDone <- err
			return
		}
		defer resp.Body.Close()
		// 会一直阻塞，直到 handler 返回或者连接被断开
		_, err = io.ReadAll(resp.Body)
		clientDone <- err
	}()

	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("请求没有到达 handler")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	err := w.Shutdown(ctx)
	if err == nil {
		t.Fatal("有连接赖着不走时，Shutdown 应返回超时错误")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("应是 ctx 超时错误，实际 %v", err)
	}

	// 关键断言：超时之后连接必须被强制断开，客户端要立刻感知到，
	// 而不是继续挂着等 handler 那个永远不返回的请求。
	select {
	case cerr := <-clientDone:
		if cerr == nil {
			t.Error("连接应当被强制断开，客户端不该正常读完")
		}
	case <-time.After(3 * time.Second):
		t.Error("Shutdown 超时后连接仍然挂着 —— 缺少 Close() 兜底")
	}
}

// 没有连接赖着时，Shutdown 应当干净利落地返回 nil
func TestShutdownReturnsCleanlyWhenIdle(t *testing.T) {
	w, addr := newTestWeb(t, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		fmt.Fprint(rw, "ok")
	}))

	resp, err := http.Get("http://" + addr + "/")
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	begin := time.Now()
	if err := w.Shutdown(ctx); err != nil {
		t.Fatalf("空闲时优雅关闭不该报错，实际 %v", err)
	}
	if elapsed := time.Since(begin); elapsed > 2*time.Second {
		t.Errorf("空闲时关闭应当很快，实际用了 %v", elapsed)
	}
}

// server 尚未启动时关闭不该 panic
func TestShutdownNilServer(t *testing.T) {
	w := &Web{}
	if err := w.Shutdown(context.Background()); err != nil {
		t.Errorf("server 为 nil 时应返回 nil，实际 %v", err)
	}
}

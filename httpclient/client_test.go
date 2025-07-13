package httpclient_test

import (
	"testing"
	"time"

	"github.com/aichy126/igo/httpclient"
	"github.com/aichy126/igo/ictx"
)

func TestHTTPClientCreation(t *testing.T) {
	// 测试客户端创建
	client := httpclient.New()
	if client == nil {
		t.Fatal("客户端创建失败")
	}

	// 测试带超时的客户端创建
	clientWithTimeout := httpclient.NewWithTimeout(5 * time.Second)
	if clientWithTimeout == nil {
		t.Fatal("带超时的客户端创建失败")
	}

	// 测试设置基础URL
	clientWithTimeout.SetBaseURL("https://api.example.com")

	t.Log("HTTP客户端创建测试通过")
}

func TestHTTPClientWithContext(t *testing.T) {
	// 创建上下文
	ctx := ictx.NewContext()
	ctx.Set("traceId", "test-trace-123")

	// 测试context创建
	if ctx == nil {
		t.Fatal("context创建失败")
	}

	traceId := ctx.GetString("traceId")
	if traceId != "test-trace-123" {
		t.Errorf("期望traceId为test-trace-123，得到%s", traceId)
	}

	t.Log("HTTP客户端上下文测试通过")
}

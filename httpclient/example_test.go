package httpclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/aichy126/igo/config"
	igocontext "github.com/aichy126/igo/context"
	"github.com/aichy126/igo/log"
)

// 测试响应结构
type TestResponse struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// 测试请求结构
type TestRequest struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func newTestClient() *Client {
	return NewClient(nil).
		WithLogging().
		WithTimeout(5*time.Second).
		WithRetry(2, 100*time.Millisecond)
}

func newTestContext() igocontext.IContext {
	ctx := igocontext.NewContext()
	ctx.Set("traceId", "test-trace-123")
	return ctx
}

func TestClient_Basic(t *testing.T) {
	conf, _ := config.NewConfig("../config/config.toml")
	log.NewLog(conf)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if traceId := r.Header.Get("X-Trace-ID"); traceId != "" {
			w.Header().Set("X-Trace-ID", traceId)
		}
		switch r.Method {
		case "GET":
			response := TestResponse{Message: "Hello World", Code: 200}
			json.NewEncoder(w).Encode(response)
		case "POST":
			var req TestRequest
			json.NewDecoder(r.Body).Decode(&req)
			response := TestResponse{Message: fmt.Sprintf("Hello %s", req.Name), Code: 201}
			json.NewEncoder(w).Encode(response)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	ctx := newTestContext()
	client := newTestClient()

	t.Run("GET请求", func(t *testing.T) {
		var response TestResponse
		err := client.GetJSON(ctx, server.URL, &response)
		if err != nil {
			t.Fatalf("GET请求失败: %v", err)
		}
		if response.Message != "Hello World" {
			t.Errorf("期望消息 'Hello World', 得到 '%s'", response.Message)
		}
	})

	t.Run("POST请求", func(t *testing.T) {
		request := TestRequest{Name: "Alice", Age: 25}
		var response TestResponse
		err := client.PostJSON(ctx, server.URL, request, &response)
		if err != nil {
			t.Fatalf("POST请求失败: %v", err)
		}
		if response.Message != "Hello Alice" {
			t.Errorf("期望消息 'Hello Alice', 得到 '%s'", response.Message)
		}
	})

	t.Run("表单请求", func(t *testing.T) {
		form := url.Values{}
		form.Set("name", "Bob")
		form.Set("age", "30")
		var response TestResponse
		err := client.PostForm(ctx, server.URL, form, &response)
		if err != nil {
			t.Fatalf("表单请求失败: %v", err)
		}
	})
}

func TestClient_Middleware(t *testing.T) {
	conf, _ := config.NewConfig("../config/config.toml")
	log.NewLog(conf)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	ctx := igocontext.NewContext()

	t.Run("超时中间件", func(t *testing.T) {
		client := NewClient(nil).WithTimeout(5 * time.Millisecond)
		_, err := client.Get(ctx, server.URL)
		if err == nil {
			t.Error("期望超时错误，但没有得到")
		}
	})

	t.Run("重试中间件", func(t *testing.T) {
		retryCount := 0
		server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			retryCount++
			if retryCount < 3 {
				http.Error(w, "Server Error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}))
		defer server2.Close()

		client := NewClient(nil).WithRetry(3, 10*time.Millisecond)
		resp, err := client.Get(ctx, server2.URL)
		if err != nil {
			t.Fatalf("重试请求失败: %v", err)
		}
		if resp.StatusCode() != http.StatusOK {
			t.Errorf("期望状态码 200, 得到 %d", resp.StatusCode())
		}
		if retryCount != 3 {
			t.Errorf("期望重试3次, 实际重试 %d 次", retryCount)
		}
	})
}

func TestClient_ResponseWrapper(t *testing.T) {
	conf, _ := config.NewConfig("../config/config.toml")
	log.NewLog(conf)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"test","code":200}`))
	}))
	defer server.Close()

	ctx := igocontext.NewContext()
	client := NewClient(nil)

	resp, err := client.Get(ctx, server.URL)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}

	body := resp.Bytes()

	response := &Response{
		Response: resp.Response,
		Body:     body,
	}

	if !response.IsSuccess() {
		t.Error("期望成功响应")
	}
	if response.IsClientError() {
		t.Error("不期望客户端错误")
	}
	if response.IsServerError() {
		t.Error("不期望服务器错误")
	}

	var data TestResponse
	if err := response.JSON(&data); err != nil {
		t.Fatalf("JSON解析失败: %v", err)
	}
	if data.Message != "test" {
		t.Errorf("期望消息 'test', 得到 '%s'", data.Message)
	}
}

// 示例：使用中间件链
func ExampleClient_Use() {
	ctx := igocontext.NewContext()
	client := NewClient(nil).
		WithLogging().
		WithTimeout(10*time.Second).
		WithRetry(3, time.Second).
		WithHeaders(map[string]string{
			"User-Agent": "IGo-HTTP-Client/1.0",
			"Accept":     "application/json",
		}).
		WithAuth("Bearer", "your-token-here")

	var response TestResponse
	err := client.GetJSON(ctx, "https://api.example.com/data", &response)
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	fmt.Printf("响应: %+v\n", response)
}

// 示例：自定义中间件
func ExampleClient_UseFunc() {
	customMiddleware := func(ctx igocontext.IContext, req *http.Request, next NextHandler) (*http.Response, error) {
		req.Header.Set("X-Request-ID", "req-12345")
		return next(ctx, req)
	}
	ctx := igocontext.NewContext()
	client := NewClient(nil).UseFunc(customMiddleware)
	resp, err := client.Get(ctx, "https://api.example.com/data")
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	fmt.Printf("状态码: %d\n", resp.StatusCode())
}

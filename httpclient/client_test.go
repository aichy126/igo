package httpclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aichy126/igo/context"
)

func TestPostJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q", r.Header.Get("Content-Type"))
		}
		var in map[string]string
		_ = json.NewDecoder(r.Body).Decode(&in)
		_ = json.NewEncoder(w).Encode(map[string]string{"echo": in["msg"]})
	}))
	defer srv.Close()

	var out struct {
		Echo string `json:"echo"`
	}
	err := New().PostJSON(t.Context(), srv.URL, map[string]string{"msg": "hello"}, &out)
	if err != nil {
		t.Fatalf("PostJSON error: %v", err)
	}
	if out.Echo != "hello" {
		t.Errorf("echo = %q, want hello", out.Echo)
	}
}

func TestGetJSONNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := New().GetJSON(t.Context(), srv.URL, nil)
	if err == nil {
		t.Fatal("非 2xx 应返回错误")
	}
}

// TestTraceIdPropagation 验证 igo context 的 meta header(traceId)自动透传
func TestTraceIdPropagation(t *testing.T) {
	var gotTrace atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTrace.Store(r.Header.Get("traceId"))
		w.WriteHeader(200)
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()

	ctx := context.NewContext()
	ctx.SetMeta("traceId", "trace-12345")

	if err := New().GetJSON(ctx, srv.URL, nil); err != nil {
		t.Fatalf("GetJSON error: %v", err)
	}
	if got := gotTrace.Load(); got != "trace-12345" {
		t.Errorf("下游收到的 traceId = %v, want trace-12345", got)
	}
}

// TestRetries 验证网络错误重试
func TestRetries(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n < 3 {
			// 前两次直接断开连接,制造网络层错误
			hj, _ := w.(http.Hijacker)
			conn, _, _ := hj.Hijack()
			conn.Close()
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	var out struct {
		Ok bool `json:"ok"`
	}
	err := New(WithRetries(3)).GetJSON(t.Context(), srv.URL, &out)
	if err != nil {
		t.Fatalf("重试后仍失败: %v", err)
	}
	if !out.Ok || calls.Load() != 3 {
		t.Errorf("ok=%v calls=%d, want true/3", out.Ok, calls.Load())
	}
}

// TestHTTPStatusNoRetry 验证 HTTP 状态码错误不触发重试
func TestHTTPStatusNoRetry(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	resp, err := New(WithRetries(3)).Get(t.Context(), srv.URL)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if resp.StatusCode != http.StatusBadGateway || calls.Load() != 1 {
		t.Errorf("status=%d calls=%d, want 502/1", resp.StatusCode, calls.Load())
	}
}

func TestTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer srv.Close()

	_, err := New(WithTimeout(100*time.Millisecond)).Get(t.Context(), srv.URL)
	if err == nil {
		t.Fatal("超时应返回错误")
	}
}

func TestDefaultHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ua := r.Header.Get("User-Agent"); ua != "igo-test" {
			t.Errorf("User-Agent = %q", ua)
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	_, err := New(WithUserAgent("igo-test")).Get(t.Context(), srv.URL)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
}

// TestReqHeaderOptions 验证单次请求 header 选项
func TestReqHeaderOptions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom") != "abc" || r.Header.Get("Accept-Language") != "zh-CN" {
			t.Errorf("缺少请求级 header: %v", r.Header)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()

	extra := http.Header{}
	extra.Set("Accept-Language", "zh-CN")
	err := New().GetJSON(t.Context(), srv.URL, nil,
		WithReqHeader("X-Custom", "abc"),
		WithReqHeaders(extra),
	)
	if err != nil {
		t.Fatalf("GetJSON error: %v", err)
	}
}

// TestGetBytes 验证 GetBytes 返回原始 body,非 2xx 报错
func TestGetBytes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/404" {
			w.WriteHeader(404)
			return
		}
		_, _ = w.Write([]byte("raw-bytes"))
	}))
	defer srv.Close()

	data, err := New().GetBytes(t.Context(), srv.URL)
	if err != nil || string(data) != "raw-bytes" {
		t.Errorf("GetBytes = %q, %v", data, err)
	}
	if _, err := New().GetBytes(t.Context(), srv.URL+"/404"); err == nil {
		t.Error("非 2xx 应返回错误")
	}
}

// TestPostFormJSON 验证表单提交 + JSON 响应解析
func TestPostFormJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		_ = json.NewEncoder(w).Encode(map[string]string{"got": r.PostForm.Get("input")})
	}))
	defer srv.Close()

	var out struct {
		Got string `json:"got"`
	}
	form := neturl.Values{}
	form.Set("input", "hello")
	err := New().PostFormJSON(t.Context(), srv.URL, form, &out)
	if err != nil || out.Got != "hello" {
		t.Errorf("PostFormJSON out=%v err=%v", out, err)
	}
}

// TestProxyOption 验证代理选项真的作用在 Transport 上
func TestProxyOption(t *testing.T) {
	c := New(WithProxyURL("http://127.0.0.1:7890"), WithInsecureSkipVerify())
	tr, ok := c.hc.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Transport 类型错误")
	}
	if tr.Proxy == nil {
		t.Error("代理未设置")
	} else if u, _ := tr.Proxy(nil); u == nil || u.Host != "127.0.0.1:7890" {
		t.Errorf("代理地址错误: %v", u)
	}
	if tr.TLSClientConfig == nil || !tr.TLSClientConfig.InsecureSkipVerify {
		t.Error("InsecureSkipVerify 未生效")
	}
}

package res

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func doRequest(t *testing.T, handler gin.HandlerFunc) (int, Body) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	handler(c)

	var body Body
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应不是合法 JSON: %v", err)
	}
	return w.Code, body
}

func TestRsucc(t *testing.T) {
	status, body := doRequest(t, func(c *gin.Context) {
		Rsucc(c, gin.H{"id": 1})
	})
	if status != 200 || body.Code != CodeOK {
		t.Errorf("Rsucc: status=%d code=%d, want 200/%d", status, body.Code, CodeOK)
	}
	if body.Data == nil {
		t.Error("Rsucc: data 不应为 nil")
	}
}

func TestRfail(t *testing.T) {
	status, body := doRequest(t, func(c *gin.Context) {
		Rfail(c, "something wrong")
	})
	if status != 200 || body.Code != CodeFail {
		t.Errorf("Rfail: status=%d code=%d, want 200/%d", status, body.Code, CodeFail)
	}
	if body.Msg != "something wrong" {
		t.Errorf("Rfail: msg=%q", body.Msg)
	}
}

func TestRfailCode(t *testing.T) {
	_, body := doRequest(t, func(c *gin.Context) {
		RfailCode(c, 401, "unauthorized")
	})
	if body.Code != 401 || body.Msg != "unauthorized" {
		t.Errorf("RfailCode: code=%d msg=%q", body.Code, body.Msg)
	}
}

func TestRlist(t *testing.T) {
	_, body := doRequest(t, func(c *gin.Context) {
		Rlist(c, 100, []string{"a", "b"})
	})
	data, ok := body.Data.(map[string]any)
	if !ok {
		t.Fatalf("Rlist: data 类型错误 %T", body.Data)
	}
	if data["total"] != float64(100) {
		t.Errorf("Rlist: total=%v, want 100", data["total"])
	}
	if items, ok := data["items"].([]any); !ok || len(items) != 2 {
		t.Errorf("Rlist: items=%v", data["items"])
	}
}

func TestSetCodes(t *testing.T) {
	SetCodes(200, -1)
	defer SetCodes(0, 1) // 恢复默认

	_, body := doRequest(t, func(c *gin.Context) {
		Rsucc(c, nil)
	})
	if body.Code != 200 {
		t.Errorf("SetCodes 后 Rsucc code=%d, want 200", body.Code)
	}
}

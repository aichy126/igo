package context

import (
	"testing"
	"time"
)

// TestGetStringSafe 验证 GetString 对各种类型都不会 panic(曾经对非 []string 值直接断言崩溃)
func TestGetStringSafe(t *testing.T) {
	ctx := NewContext()

	ctx.Set("str", "hello")
	if got := ctx.GetString("str"); got != "hello" {
		t.Errorf("GetString(str) = %q, want %q", got, "hello")
	}

	ctx.Set("slice", []string{"first", "second"})
	if got := ctx.GetString("slice"); got != "first" {
		t.Errorf("GetString(slice) = %q, want %q", got, "first")
	}

	ctx.Set("empty-slice", []string{})
	if got := ctx.GetString("empty-slice"); got != "" {
		t.Errorf("GetString(empty-slice) = %q, want empty", got)
	}

	ctx.Set("int64", int64(123))
	if got := ctx.GetString("int64"); got != "123" {
		t.Errorf("GetString(int64) = %q, want %q", got, "123")
	}

	if got := ctx.GetString("not-exist"); got != "" {
		t.Errorf("GetString(not-exist) = %q, want empty", got)
	}
}

// TestSetGetCaseInsensitive 验证 key 大小写不敏感的读写
func TestSetGetCaseInsensitive(t *testing.T) {
	ctx := NewContext()
	ctx.Set("user_id", int64(42))

	if got := ctx.GetInt64("user_id"); got != 42 {
		t.Errorf("GetInt64(user_id) = %d, want 42", got)
	}
	if got := ctx.GetInt64("USER_ID"); got != 42 {
		t.Errorf("GetInt64(USER_ID) = %d, want 42", got)
	}

	// 通过标准 context.Context 接口读取
	if got := ctx.Value("user_id"); got != int64(42) {
		t.Errorf("Value(user_id) = %v, want 42", got)
	}
}

// TestLogNotPanic 验证 LogInfo/LogError 在任何情况下不 panic(日志未初始化时降级为控制台)
func TestLogNotPanic(t *testing.T) {
	ctx := NewContext()
	ctx.Set("traceId", "test-trace-id")
	ctx.LogInfo("info message")
	ctx.LogError("error message")
}

// TestWithTimeout 验证派生 context 不影响原 context
func TestWithTimeout(t *testing.T) {
	ctx := NewContext()
	ctx.Set("key", "value")

	newCtx, cancel := ctx.WithTimeout(time.Millisecond * 10)
	defer cancel()

	if got := newCtx.GetString("key"); got != "value" {
		t.Errorf("derived ctx GetString(key) = %q, want %q", got, "value")
	}

	<-newCtx.Done()
	if newCtx.Err() == nil {
		t.Error("derived ctx should be timed out")
	}
	if ctx.Err() != nil {
		t.Error("original ctx should not be affected by derived timeout")
	}
}

// TestRepeatedSetNoLeak 验证反复 Set 同一个 key 不会让 context 链无限增长
func TestRepeatedSetNoLeak(t *testing.T) {
	ctx := NewContext()
	for i := 0; i < 10000; i++ {
		ctx.Set("counter", i)
	}
	if got := ctx.GetInt("counter"); got != 9999 {
		t.Errorf("GetInt(counter) = %d, want 9999", got)
	}
}

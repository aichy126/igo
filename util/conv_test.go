package util

import (
	"testing"
	"time"
)

func TestString(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{"abc", "abc"},
		{123, "123"},
		{int64(456), "456"},
		{3.14, "3.14"},
		{true, "true"},
		{nil, ""},
		{[]byte("bytes"), "bytes"},
		// Stringer 类型保持旧版行为:输出 String() 结果而不是数字
		{time.July, "July"},
	}
	for _, tc := range cases {
		if got := String(tc.in); got != tc.want {
			t.Errorf("String(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestToInt64(t *testing.T) {
	cases := []struct {
		in      any
		want    int64
		wantErr bool
	}{
		{123, 123, false},
		{"456", 456, false},
		{"3.99", 3, false},
		{" 42 ", 42, false},
		{"", 0, false},
		{3.7, 3, false},
		{true, 1, false},
		{nil, 0, false},
		{"abc", 0, true},
		{[]string{"x"}, 0, true},
	}
	for _, tc := range cases {
		got, err := ToInt64(tc.in)
		if (err != nil) != tc.wantErr {
			t.Errorf("ToInt64(%v) err = %v, wantErr=%v", tc.in, err, tc.wantErr)
			continue
		}
		if got != tc.want {
			t.Errorf("ToInt64(%v) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestToUint(t *testing.T) {
	if v, err := ToUint("99"); err != nil || v != 99 {
		t.Errorf("ToUint(99) = %d, %v", v, err)
	}
	if _, err := ToUint(-1); err == nil {
		t.Error("负数转无符号应报错")
	}
}

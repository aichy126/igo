package util

import "testing"

func TestPageQueryNormalize(t *testing.T) {
	cases := []struct {
		name               string
		page, size         int
		wantPage, wantSize int
		wantOffset         int
	}{
		{"默认值", 0, 0, 1, 20, 0},
		{"负数页码", -1, 10, 1, 10, 0},
		{"正常翻页", 3, 10, 3, 10, 20},
		{"超出上限", 1, 9999, 1, 100, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := PageQuery{Page: tc.page, PageSize: tc.size}
			p.Normalize(20, 100)
			if p.Page != tc.wantPage || p.PageSize != tc.wantSize {
				t.Errorf("Normalize: got page=%d size=%d, want %d/%d", p.Page, p.PageSize, tc.wantPage, tc.wantSize)
			}
			if got := p.Offset(); got != tc.wantOffset {
				t.Errorf("Offset: got %d, want %d", got, tc.wantOffset)
			}
		})
	}
}

func TestSafeOrderBy(t *testing.T) {
	allow := map[string]string{
		"created": "created_at desc",
		"name":    "username asc",
	}
	if got := SafeOrderBy("created", allow, "id desc"); got != "created_at desc" {
		t.Errorf("SafeOrderBy 命中白名单: got %q", got)
	}
	if got := SafeOrderBy("1;drop table users", allow, "id desc"); got != "id desc" {
		t.Errorf("SafeOrderBy 未命中应返回默认值: got %q", got)
	}
}

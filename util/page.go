package util

// PageQuery 通用分页参数，内嵌进各业务的 Search 结构复用。
//
//	type NoteSearch struct {
//	    util.PageQuery
//	    Keyword string `form:"keyword"`
//	}
type PageQuery struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"page_size" form:"page_size"`
}

// Normalize 归一分页参数：Page 从 1 起；PageSize 缺省用 defSize，并 clamp 到 [1, maxSize]。
// 调用 Offset 前先 Normalize。
func (p *PageQuery) Normalize(defSize, maxSize int) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 {
		p.PageSize = defSize
	}
	if p.PageSize > maxSize {
		p.PageSize = maxSize
	}
}

// Offset 返回 SQL 偏移量（需先 Normalize）。
func (p *PageQuery) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// SafeOrderBy 按白名单把外部排序字段映射到真实「列名 [方向]」，防 SQL 注入。
// allow 形如 {"created": "created_at desc", "name": "username asc"}；
// input 命中返回映射值，否则返回 def。
func SafeOrderBy(input string, allow map[string]string, def string) string {
	if v, ok := allow[input]; ok {
		return v
	}
	return def
}

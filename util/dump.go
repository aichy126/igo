package util

import (
	"github.com/davecgh/go-spew/spew"
)

// Dump 调试输出任意值(带类型和结构信息),仅用于本地开发调试
func Dump(vs ...any) {
	spew.Dump(vs...)
}

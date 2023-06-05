package util

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
)

func Dump(a ...interface{}) {
	spew.Dump(a)
}

// CDump 带颜色输出
func CDump(a ...interface{}) {
	fmt.Print("\x1b[33;44m")                  //设置颜色样式
	fmt.Print("==========[dump]==========\n") //设置颜色样式
	spew.Dump(a)
	fmt.Print("==========================") //设置颜色样式
	fmt.Print("\x1b[0m\n")                  //样式结束符,清楚之前的显示属性
}

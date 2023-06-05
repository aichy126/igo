package main

import (
	"github.com/aichy126/igo"
	"github.com/aichy126/igo/util"
)

func main() {
	igo.App = igo.NewApp("") //初始化各个组件
	debug := util.ConfGetbool("local.debug")
	util.CDump(debug)

}

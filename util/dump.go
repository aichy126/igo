package util

import (
	"os"

	"github.com/gookit/goutil/dump"
)

const defaultSkip = 3

var (
	std = dump.NewDumper(os.Stdout, defaultSkip)
)

func Dump(vs ...any) {
	std.Print(vs...)
}

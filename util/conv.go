package util

import (
	"github.com/gookit/goutil/mathutil"
	"github.com/gookit/goutil/strutil"
)

// Bool convert value to bool

// String always convert value to string, will ignore error
func String(v any) string {
	s, _ := strutil.AnyToString(v, false)
	return s
}

// ToString convert value to string, will return error on fail.
func ToString(v any) (string, error) {
	return strutil.AnyToString(v, true)
}

// Int convert value to int
func Int(v any) int {
	iv, _ := mathutil.ToInt(v)
	return iv
}

// ToInt try to convert value to int
func ToInt(v any) (int, error) {
	return mathutil.ToInt(v)
}

// Int64 convert value to int64
func Int64(v any) int64 {
	iv, _ := mathutil.ToInt64(v)
	return iv
}

// ToInt64 try to convert value to int64
func ToInt64(v any) (int64, error) {
	return mathutil.ToInt64(v)
}

// Uint convert value to uint64
func Uint(v any) uint64 {
	iv, _ := mathutil.ToUint(v)
	return iv
}

// ToUint try to convert value to uint64
func ToUint(v any) (uint64, error) {
	return mathutil.ToUint(v)
}

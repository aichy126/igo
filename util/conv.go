package util

import (
	"github.com/spf13/cast"
)

// Bool convert value to bool

// String always convert value to string, will ignore error
func String(v any) string {
	return ToString(v)
}

// ToString convert value to string, will return error on fail.
func ToString(v any) string {
	return cast.ToString(v)
}

// Int convert value to int
func Int(v any) int {
	return cast.ToInt(v)
}

// ToInt try to convert value to int
func ToInt(v any) int {
	return cast.ToInt(v)
}

// Int64 convert value to int64
func Int64(v any) int64 {
	return cast.ToInt64(v)
}

// ToInt64 try to convert value to int64
func ToInt64(v any) int64 {
	return cast.ToInt64(v)
}

// Uint convert value to uint64
func Uint(v any) uint64 {
	return cast.ToUint64(v)
}

// ToUint try to convert value to uint64
func ToUint(v any) uint64 {
	return cast.ToUint64(v)
}

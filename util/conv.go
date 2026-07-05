package util

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// String always convert value to string, will ignore error
func String(v any) string {
	s, _ := ToString(v)
	return s
}

// ToString convert value to string, will return error on fail.
func ToString(v any) (string, error) {
	switch val := v.(type) {
	case nil:
		return "", nil
	case string:
		return val, nil
	case []byte:
		return string(val), nil
	case bool:
		return strconv.FormatBool(val), nil
	case int:
		return strconv.Itoa(val), nil
	case int8:
		return strconv.FormatInt(int64(val), 10), nil
	case int16:
		return strconv.FormatInt(int64(val), 10), nil
	case int32:
		return strconv.FormatInt(int64(val), 10), nil
	case int64:
		return strconv.FormatInt(val, 10), nil
	case uint:
		return strconv.FormatUint(uint64(val), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(val), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(val), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(val), 10), nil
	case uint64:
		return strconv.FormatUint(val, 10), nil
	case float32:
		return strconv.FormatFloat(float64(val), 'f', -1, 32), nil
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64), nil
	case error:
		return val.Error(), nil
	case fmt.Stringer:
		// 保持与旧版一致:time.Month 等 Stringer 类型输出其 String() 结果(如 "July")
		return val.String(), nil
	default:
		return fmt.Sprint(v), nil
	}
}

// Int convert value to int
func Int(v any) int {
	iv, _ := ToInt(v)
	return iv
}

// ToInt try to convert value to int
func ToInt(v any) (int, error) {
	iv, err := ToInt64(v)
	return int(iv), err
}

// Int64 convert value to int64
func Int64(v any) int64 {
	iv, _ := ToInt64(v)
	return iv
}

// ToInt64 try to convert value to int64
func ToInt64(v any) (int64, error) {
	switch val := v.(type) {
	case nil:
		return 0, nil
	case int:
		return int64(val), nil
	case int8:
		return int64(val), nil
	case int16:
		return int64(val), nil
	case int32:
		return int64(val), nil
	case int64:
		return val, nil
	case uint:
		return int64(val), nil
	case uint8:
		return int64(val), nil
	case uint16:
		return int64(val), nil
	case uint32:
		return int64(val), nil
	case uint64:
		return int64(val), nil
	case float32:
		return int64(val), nil
	case float64:
		return int64(val), nil
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil
	case time.Duration:
		return int64(val), nil
	case string:
		s := strings.TrimSpace(val)
		if s == "" {
			return 0, nil
		}
		if iv, err := strconv.ParseInt(s, 10, 64); err == nil {
			return iv, nil
		}
		// 兼容 "3.14" 这类数字字符串
		if fv, err := strconv.ParseFloat(s, 64); err == nil {
			return int64(fv), nil
		}
		return 0, fmt.Errorf("无法把字符串 %q 转换为整数", val)
	default:
		return 0, fmt.Errorf("无法把 %T 转换为整数", v)
	}
}

// Uint convert value to uint64
func Uint(v any) uint64 {
	iv, _ := ToUint(v)
	return iv
}

// ToUint try to convert value to uint64
func ToUint(v any) (uint64, error) {
	iv, err := ToInt64(v)
	if err != nil {
		return 0, err
	}
	if iv < 0 {
		return 0, fmt.Errorf("负数 %d 无法转换为无符号整数", iv)
	}
	return uint64(iv), nil
}

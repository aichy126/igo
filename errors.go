package igo

import (
	"fmt"
	"net/http"
)

// ErrorCode 错误码类型
type ErrorCode int

const (
	// 系统级错误码 (1000-1999)
	ErrCodeInternalServer ErrorCode = 1000
	ErrCodeConfigInvalid  ErrorCode = 1001
	ErrCodeDatabaseError  ErrorCode = 1002
	ErrCodeCacheError     ErrorCode = 1003
	ErrCodeNetworkError   ErrorCode = 1004

	// 业务级错误码 (2000-2999)
	ErrCodeInvalidParam    ErrorCode = 2000
	ErrCodeNotFound        ErrorCode = 2001
	ErrCodeUnauthorized    ErrorCode = 2002
	ErrCodeForbidden       ErrorCode = 2003
	ErrCodeConflict        ErrorCode = 2004
	ErrCodeTooManyRequests ErrorCode = 2005
)

// AppError 应用错误
type AppError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
	Err     error     `json:"-"`
}

// Error 实现error接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 返回原始错误
func (e *AppError) Unwrap() error {
	return e.Err
}

// HTTPStatus 返回对应的HTTP状态码
func (e *AppError) HTTPStatus() int {
	switch e.Code {
	case ErrCodeInvalidParam:
		return http.StatusBadRequest
	case ErrCodeNotFound:
		return http.StatusNotFound
	case ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case ErrCodeForbidden:
		return http.StatusForbidden
	case ErrCodeConflict:
		return http.StatusConflict
	case ErrCodeTooManyRequests:
		return http.StatusTooManyRequests
	case ErrCodeConfigInvalid, ErrCodeDatabaseError, ErrCodeCacheError, ErrCodeNetworkError:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// NewError 创建新的应用错误
func NewError(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// NewErrorWithDetails 创建带详细信息的应用错误
func NewErrorWithDetails(code ErrorCode, message, details string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// WrapError 包装现有错误
func WrapError(code ErrorCode, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// 预定义错误
var (
	ErrInternalServer = NewError(ErrCodeInternalServer, "内部服务器错误")
	ErrConfigInvalid  = NewError(ErrCodeConfigInvalid, "配置无效")
	ErrDatabaseError  = NewError(ErrCodeDatabaseError, "数据库错误")
	ErrCacheError     = NewError(ErrCodeCacheError, "缓存错误")
	ErrNetworkError   = NewError(ErrCodeNetworkError, "网络错误")

	ErrInvalidParam    = NewError(ErrCodeInvalidParam, "参数无效")
	ErrNotFound        = NewError(ErrCodeNotFound, "资源不存在")
	ErrUnauthorized    = NewError(ErrCodeUnauthorized, "未授权")
	ErrForbidden       = NewError(ErrCodeForbidden, "禁止访问")
	ErrConflict        = NewError(ErrCodeConflict, "资源冲突")
	ErrTooManyRequests = NewError(ErrCodeTooManyRequests, "请求过于频繁")
)

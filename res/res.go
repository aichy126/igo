// Package res 提供统一的 HTTP JSON 响应格式:{"code": .., "msg": .., "data": ..}
// HTTP 状态码恒为 200,业务成功/失败通过 code 字段区分。
//
// 默认 code:成功=0,失败=1。如果项目使用其他约定(如成功=200),
// 可在 main 初始化时调用 res.SetCodes(200, 1) 全局调整。
package res

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

var (
	// CodeOK 业务成功码
	CodeOK = 0
	// CodeFail 通用业务失败码
	CodeFail = 1
)

// SetCodes 全局调整成功/失败业务码(在 main 初始化时调用一次)
func SetCodes(ok, fail int) {
	CodeOK = ok
	CodeFail = fail
}

// Body 统一响应结构
type Body struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

// Rsucc 返回成功响应
func Rsucc(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Body{Code: CodeOK, Msg: "success", Data: data})
}

// Rfail 返回失败响应(业务码 = CodeFail)
func Rfail(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, Body{Code: CodeFail, Msg: msg, Data: nil})
}

// RfailCode 返回带自定义业务码的失败响应
func RfailCode(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, Body{Code: code, Msg: msg, Data: nil})
}

// Rlist 返回分页列表响应:data = { "total": total, "items": items }
func Rlist(c *gin.Context, total int64, items any) {
	c.JSON(http.StatusOK, Body{Code: CodeOK, Msg: "success", Data: gin.H{"total": total, "items": items}})
}

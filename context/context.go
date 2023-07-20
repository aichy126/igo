package context

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"time"

	"github.com/aichy126/igo/log"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const CommonContextKey = "context"

// http 的header
const (
	HeaderGinContextKey = "gin-context"
	HttpRequestKey      = "Http-Request"
)

type Context = IContext
type IContext interface {
	context.Context
	Set(key string, value interface{})
	// SetMeta与Set的区别是 SetMeta设置的key在ctx.GetHeaderMap()时会返回
	// 而这个map通常是ddns client请求别的服务时需要透传过去的Header
	// 故如果你觉得这个key需要进行透传，请用SetMeta
	SetMeta(key string, value string)
	WithValue(key interface{}, val interface{}) IContext
	WithCancel() (IContext, CancelFunc)
	WithTimeout(d time.Duration) (IContext, CancelFunc)

	GetHeaders() http.Header

	Get(key string) (value interface{}, exists bool)

	GetString(key string) (v string)
	GetInt64(key string) (i int64)
	GetInt(key string) (i int)

	GetGoContext() context.Context
	GetAllKey() map[string]interface{}
	GetHttpRequest() *http.Request
	Info(msg string, fields ...log.Field)
}
type contextImpl struct {
	context.Context
	Keys map[string]interface{}
	// 用户记录keys中哪些key 作为header使用，当进行ddns请求时，这些header将被序列化后传递到对端
	meta map[string]string

	lock sync.RWMutex
}

func Background() IContext {
	return NewContext()
}
func TODO() IContext {
	return NewContext()
}

type CancelFunc = context.CancelFunc

var DeadlineExceeded = context.DeadlineExceeded
var Canceled = context.Canceled
var _ IContext = &contextImpl{}

func WithContext(ctx context.Context) *contextImpl {
	c := &contextImpl{
		Context: ctx,
		Keys:    make(map[string]interface{}),
		meta:    make(map[string]string),
	}
	return c
}

func NewContext() IContext {
	return WithContext(context.Background())
}

func NewContextWithGinHeader(c *gin.Context) IContext {
	ctx := WithContext(context.Background())
	//继承gin header
	for k, v := range c.Request.Header {
		ctx.Set(k, v)
	}
	//继承gin traceId
	traceId := c.GetString("traceId")
	if traceId != "" {
		ctx.Set("traceId", traceId)
	} else {
		ctx.Set("traceId", uuid.New().String())
	}
	return ctx
}

func WithCancel(parent context.Context) (IContext, CancelFunc) {
	switch p := parent.(type) {
	case IContext:
		return p.WithCancel()
	default:
		c, cancel := context.WithCancel(p)
		return WithContext(c), cancel
	}
}

func WithTimeout(goctx context.Context, d time.Duration) (IContext, CancelFunc) {
	switch p := goctx.(type) {
	case IContext:
		return p.WithTimeout(d)
	default:
		c, cancel := context.WithTimeout(p, d)
		return WithContext(c), cancel
	}
}

func WithValue(parent context.Context, key interface{}, val interface{}) IContext {
	switch p := parent.(type) {
	case IContext:
		return p.WithValue(key, val)
	default:
		return WithContext(context.WithValue(p, key, val))
	}
}

func (c *contextImpl) WithContext(goctx context.Context) IContext {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.Context = goctx
	return c
}

func (c *contextImpl) WithCancel() (IContext, CancelFunc) {
	c.lock.RLock()
	goctxWithCancel, cancel := context.WithCancel(c.Context)
	newCtx := c.clone()
	c.lock.RUnlock()
	newCtx.Context = goctxWithCancel
	return newCtx, cancel
}

// Withtime 会创建一个新的Context返回， 以避免直接将cancelContext链到原context c上，对其继续使用导致干扰
func (c *contextImpl) WithTimeout(d time.Duration) (IContext, CancelFunc) {
	c.lock.RLock()
	goctxWithCancel, cancel := context.WithTimeout(c.Context, d)
	newCtx := c.clone()
	c.lock.RUnlock()
	newCtx.Context = goctxWithCancel
	return newCtx, cancel

}

func (ctx *contextImpl) GetHeaders() http.Header {
	header := http.Header{}
	return header
}

// 重新实现context.Context 的Value()接口
func (c *contextImpl) Value(key interface{}) interface{} {
	keyString, ok := key.(string)
	if ok {
		value, ok := c.Get(keyString)
		if ok {
			return value
		}
	}
	return c.Context.Value(key)
}

func (c *contextImpl) Clone() IContext {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.clone()
}

func (c *contextImpl) clone() *contextImpl {
	newCtx := &contextImpl{
		Context: c.Context,
		Keys:    make(map[string]interface{}),
		meta:    make(map[string]string),
	}
	for key, value := range c.Keys {
		newCtx.Keys[key] = value
	}
	for key, value := range c.meta {
		newCtx.meta[key] = value
	}
	return newCtx
}

// Set is used to store a new key/value pair exclusively for this context.
// It also lazy initializes  c.Keys if it was not used previously.
func (c *contextImpl) Set(key string, value interface{}) {
	c.lock.Lock()
	if c.Keys == nil {
		c.Keys = make(map[string]interface{})
	}
	c.Keys[strings.ToUpper(key)] = value
	c.Context = context.WithValue(c.Context, key, value)
	c.lock.Unlock()
}

func (c *contextImpl) WithValue(key interface{}, val interface{}) IContext {
	switch k := key.(type) {
	case string:
		c.Set(k, val)
	default:
		c.lock.Lock()
		c.Context = context.WithValue(c.Context, key, val)
		c.lock.Unlock()
	}
	return c
}

func (c *contextImpl) SetMeta(key string, value string) {
	c.lock.Lock()
	if c.Keys == nil {
		c.Keys = make(map[string]interface{})
	}
	c.Keys[strings.ToUpper(key)] = value
	if c.meta == nil {
		c.meta = make(map[string]string)
	}
	c.meta[key] = value
	c.Context = context.WithValue(c.Context, key, value)

	c.lock.Unlock()
}

// Get returns the value for the given key, ie: (value, true).
// If the value does not exists it returns (nil, false)
func (c *contextImpl) Get(key string) (value interface{}, exists bool) {
	c.lock.RLock()
	value, exists = c.get(key)
	c.lock.RUnlock()
	return
}

func (c *contextImpl) get(key string) (value interface{}, exists bool) {
	value, exists = c.Keys[strings.ToUpper(key)]
	if exists {
		return
	}
	value = c.Context.Value(key)
	if value != nil {
		return value, true
	}
	return nil, false
}

func (c *contextImpl) GetString(key string) (v string) {
	c.lock.RLock()
	v, _ = c.getString(key)
	c.lock.RUnlock()
	return
}

func (c *contextImpl) getString(key string) (v string, exists bool) {
	obj, exists := c.get(key)
	if !exists {
		return
	}
	return fmt.Sprintf("%v", obj.([]string)[0]), exists
}

// GetInt returns the value associated with the key as an integer.
func (c *contextImpl) GetInt(key string) (i int) {
	if val, ok := c.Get(key); ok && val != nil {
		i, ok = val.(int)
		if ok {
			return i
		}
		valStr, ok := val.(string)
		if !ok {
			return
		}
		i, _ = strconv.Atoi(valStr)
	}
	return
}

func (c *contextImpl) GetInt64(key string) (i int64) {
	if val, ok := c.Get(key); ok && val != nil {
		i, ok = val.(int64)
		if ok {
			return i
		}

		valStr, ok := val.(string)
		if !ok {
			return
		}
		i, _ = strconv.ParseInt(valStr, 10, 0)
	}
	return
}

func (ctx *contextImpl) GetGoContext() context.Context {
	return ctx
}

func (ctx *contextImpl) GetAllKey() map[string]interface{} {
	ctx.lock.RLock()
	defer ctx.lock.RUnlock()
	retMap := make(map[string]interface{}, 0)
	for key, value := range ctx.Keys {
		retMap[key] = value
		//retMap[key] = value.([]string)[0]
	}
	return retMap
}

func (ctx *contextImpl) GetHttpRequest() *http.Request {
	obj, ok := ctx.Get(HttpRequestKey)
	if !ok {
		return nil
	}
	req, ok := obj.(*http.Request)
	if !ok {
		return nil
	}
	return req
}

func (ctx *contextImpl) Info(msg string, fields ...log.Field) {
	zap.AddCallerSkip(2)
	traceId, has := ctx.get("traceId")
	if has {
		fields = append(fields, log.Any("traceId", traceId))
	}
	log.Info(msg, fields...)
}

type IGetter interface {
	//为了去除去gin.Context的直接依赖
	Get(string) (value interface{}, exists bool)
}

func Ginform(c IGetter) IContext {
	if c == nil {
		log.Error("common.Transform c_is_nil", log.Any("c", c))
		return NewContext()
	}
	if ic, ok := c.(IContext); ok {
		return ic
	}
	if ginctx, ok := c.(*gin.Context); ok {
		return NewContextWithGinHeader(ginctx)
	}
	return NewContext()
}

package util

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"sync"
	"time"

	"github.com/aichy126/igo/log"
)

// 用法 通常此函数放在defer 之后 ，形如: defer Recover()
// 通常是这样用
//
//	go func(){
//	    defer Recover()
//	    dosomething()
//	}()
func Recover() {
	err := recover()
	if err != nil {
		errTraceId := fmt.Sprintf("%x", RandomID())

		log.Error("panic recoverd with err", log.Any("error", err), log.Any("stackid", errTraceId))

		for i := 1; i < 500; i++ {
			_, file, line, ok := runtime.Caller(i)
			if ok {
				log.Error("panic recoverd with err", log.Any("stack_index", i), log.Any("stackid", errTraceId), log.Any(file, line))
			} else {
				break
			}
		}
	}
}

func GoroutineFunc(fun interface{}, args ...interface{}) (err error) {
	v := reflect.ValueOf(fun)
	go func(err error) {
		defer Recover()
		switch v.Kind().String() {
		case "func":
			pps := make([]reflect.Value, 0, len(args))
			for _, arg := range args {
				pps = append(pps, reflect.ValueOf(arg))
			}
			v.Call(pps)
		default:
			err = errors.New(fmt.Sprintf("func is not func,type=%v", v.Kind().String()))
			log.Error("GoroutineFunc", log.Any("error", err))
		}
	}(err)
	return
}

var (
	seededIDGen = rand.New(rand.NewSource(time.Now().UnixNano()))
	// The golang rand generators are *not* intrinsically thread-safe.
	seededIDLock sync.Mutex
)

func RandomID() uint64 {
	seededIDLock.Lock()
	defer seededIDLock.Unlock()
	return uint64(seededIDGen.Int63())
}

func RandomIDInt64() int64 {
	seededIDLock.Lock()
	defer seededIDLock.Unlock()
	return seededIDGen.Int63()
}

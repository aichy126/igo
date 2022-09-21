package httpclient

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	gocontext "context"

	"github.com/aichy126/igo/context"
)

// 不用重用HTTPRequest对像，也不要用任何变量永久存储*HTTPRequest对象
// 如果你想重用，请重用HttpSettings
// setting:=NewHttpSettings().SetDefaultTimeout(timeout).SetUserAgent("ddns")
// httpRequest:=setting.Request(url,"GET")
// strResult,err:=httpRequest.String(ctx)
// strResult,err:=httpRequest.Byts(ctx)
// strResult,err:=httpRequest.httpRequest(ctx)

var (
	defaultSetting = HttpSettings{
		timeout:  60 * time.Second,
		gzip:     true, // 如果对端传过来的数据是以gzip压缩的， 我们是否尝试对之解密
		dumpBody: true, // 当Debug(true)时，dump request时，是否也同时dump request的body,httputil.DumpRequest(req,bool）,
		debug:    false,
	}

	defaultSettingMutex sync.RWMutex
	defaultCookieJar, _ = cookiejar.New(nil)
)

type HttpSettings struct {
	mutex           sync.RWMutex
	debug           bool
	userAgent       string
	timeout         time.Duration
	tLSClientConfig *tls.Config
	proxy           func(*http.Request) (*url.URL, error)
	transport       http.RoundTripper
	checkRedirect   func(req *http.Request, via []*http.Request) error
	enableCookie    bool
	gzip            bool
	dumpBody        bool
	retries         int
	client          *http.Client
}

func NewHttpSettings() *HttpSettings {
	setting := &HttpSettings{}
	*setting = getDefaultSetting()
	setting.transport = nil
	setting.client = nil

	return setting
}

// Get returns *HttpRequest with GET method.
func (setting *HttpSettings) ContextRequestGet(ctx gocontext.Context, url string) *HttpRequest {
	return setting.ContextRequest(ctx, url, "GET")
}

// Post returns *HttpRequest with POST method.
func (setting *HttpSettings) ContextRequestPost(ctx gocontext.Context, url string) *HttpRequest {
	return setting.ContextRequest(ctx, url, "POST")
}

// Put returns *HttpRequest with PUT method.
func (setting *HttpSettings) ContextRequestPut(ctx gocontext.Context, url string) *HttpRequest {
	return setting.ContextRequest(ctx, url, "PUT")
}
func (setting *HttpSettings) ContextRequestPatch(ctx gocontext.Context, url string) *HttpRequest {
	return setting.ContextRequest(ctx, url, "PATCH")
}

// Delete returns *HttpRequest DELETE method.
func (setting *HttpSettings) ContextRequestDelete(ctx gocontext.Context, url string) *HttpRequest {
	return setting.ContextRequest(ctx, url, "DELETE")
}

// Head returns *HttpRequest with HEAD method.
func (setting *HttpSettings) ContextRequestHead(ctx gocontext.Context, url string) *HttpRequest {
	return setting.ContextRequest(ctx, url, "HEAD")
}

func (setting *HttpSettings) ContextRequest(ctx gocontext.Context, rawurl, method string) *HttpRequest {
	req := setting.request(ctx, rawurl, method)
	ictx, ok := ctx.(context.IContext)
	if ok {
		req.Headers(ictx.GetHeaders())
	}
	return req
}

// HttpSettings创建的一个HTTPRequest表示一个http请求， 不要用同一个HttpRequest进行多次请求
// Get returns *HttpRequest with GET method.
func (setting *HttpSettings) RequestGet(url string) *HttpRequest {
	return setting.Request(url, "GET")
}

// Post returns *HttpRequest with POST method.
func (setting *HttpSettings) RequestPost(url string) *HttpRequest {
	return setting.Request(url, "POST")
}

// Put returns *HttpRequest with PUT method.
func (setting *HttpSettings) RequestPut(url string) *HttpRequest {
	return setting.Request(url, "PUT")
}
func (setting *HttpSettings) RequestPatch(url string) *HttpRequest {
	return setting.Request(url, "PATCH")
}

// Delete returns *HttpRequest DELETE method.
func (setting *HttpSettings) RequestDelete(url string) *HttpRequest {
	return setting.Request(url, "DELETE")
}

// Head returns *HttpRequest with HEAD method.
func (setting *HttpSettings) RequestHead(url string) *HttpRequest {
	return setting.Request(url, "HEAD")
}

func (setting *HttpSettings) Request(rawurl, method string) *HttpRequest {
	return setting.request(nil, rawurl, method)
}
func (setting *HttpSettings) request(ctx gocontext.Context, rawurl, method string) *HttpRequest {
	//  TODO: use sync.Pool
	httpReq := &HttpRequest{
		method:  strings.ToUpper(method),
		url:     rawurl,
		header:  nil,
		params:  nil,
		files:   nil,
		retries: setting.retries,
	}
	if ctx == nil {
		ctx = context.NewContext()
	}
	httpReq.ctx = ctx

	setting.checkClient()
	setting.mutex.RLock()
	httpReq.setting = *setting
	setting.mutex.RUnlock()
	return httpReq

}

func (setting *HttpSettings) checkClient() {
	setting.mutex.RLock()
	if setting.client != nil {
		setting.mutex.RUnlock()
		return
	}

	trans := setting.transport
	timeout := time.Duration(setting.timeout)
	if trans == nil {
		httpTrans := getTransport(timeout)
		if setting.tLSClientConfig != nil {
			httpTrans.TLSClientConfig = setting.tLSClientConfig
		}
		if setting.proxy != nil {
			httpTrans.Proxy = setting.proxy
		}
		trans = httpTrans
	} else {
		if t, ok := trans.(*http.Transport); ok {
			if t.TLSClientConfig == nil {
				t.TLSClientConfig = setting.tLSClientConfig
			}
			if t.Proxy == nil {
				t.Proxy = setting.proxy
			}
		}
	}

	var jar http.CookieJar
	if setting.enableCookie {
		jar = defaultCookieJar
	}

	setting.mutex.RUnlock()

	setting.mutex.Lock()
	if setting.client == nil {
		setting.client = &http.Client{
			Transport: trans,
			Jar:       jar,
			// Timeout:       timeout,
			CheckRedirect: setting.checkRedirect,
		}
	} else {
		setting.client.Transport = trans
		setting.client.Jar = jar
		setting.client.CheckRedirect = setting.checkRedirect
	}
	setting.mutex.Unlock()

}

// SetEnableCookie sets enable/disable cookiejar
func (setting *HttpSettings) SetEnableCookie(enable bool) *HttpSettings {
	setting.mutex.Lock()
	if enable != setting.enableCookie {
		setting.enableCookie = enable
		if setting.client != nil {
			if enable {
				setting.client.Jar = defaultCookieJar
			} else {
				setting.client.Jar = nil
			}

		}

	}
	setting.mutex.Unlock()
	return setting
}

// SetUserAgent sets User-Agent header field
func (setting *HttpSettings) SetUserAgent(useragent string) *HttpSettings {
	setting.mutex.Lock()
	setting.userAgent = useragent
	setting.mutex.Unlock()
	return setting
}

// Debug sets show debug or not when executing request.
func (setting *HttpSettings) Debug(isdebug bool) *HttpSettings {
	setting.mutex.Lock()
	setting.debug = isdebug
	setting.mutex.Unlock()
	return setting
}

// Retries sets Retries times.
// default is 0 means no retried.
// -1 means retried forever.
// others means retried times.
func (setting *HttpSettings) SetDefaultRetries(times int) *HttpSettings {
	if times < 0 {
		return setting
	}

	setting.mutex.Lock()
	setting.retries = times
	setting.mutex.Unlock()
	return setting
}

// SetDefaultTimeout sets connect time out and read-write time out for Request.
func (setting *HttpSettings) SetDefaultTimeout(timeout time.Duration) *HttpSettings {
	setting.mutex.Lock()
	if timeout != setting.timeout {
		setting.timeout = timeout
		if setting.transport != nil {
			if t, ok := setting.transport.(*http.Transport); ok {
				setTransportTimeout(t, timeout)
			} else {
				setting.transport = nil // 当属性发生变更时setting.transport 需要在 checkClient 重新生成
			}
		}

		setting.client = nil
		// if setting.client != nil {
		// 	// setting.client.Timeout = timeout
		// }
	}
	setting.mutex.Unlock()
	return setting
}

// SetTLSClientConfig sets tls connection configurations if visiting https url.
func (setting *HttpSettings) SetTLSClientConfig(config *tls.Config) *HttpSettings {
	setting.mutex.Lock()
	setting.tLSClientConfig = config

	if setting.transport != nil {
		if t, ok := setting.transport.(*http.Transport); ok {
			if t.TLSClientConfig == nil {
				t.TLSClientConfig = setting.tLSClientConfig
			}
		} else {
			setting.client = nil // 当属性发生变更时setting.client 需要重新生成
		}
	} else {
		setting.client = nil // 当属性发生变更时setting.client 需要重新生成
	}

	setting.mutex.Unlock()
	return setting
}

// SetTransport set the setting transport
func (setting *HttpSettings) SetTransport(transport http.RoundTripper) *HttpSettings {
	setting.mutex.Lock()
	setting.transport = transport
	if setting.client != nil {
		setting.client.Transport = transport // 当属性发生变更时setting.client 需要重新生成
	}

	setting.mutex.Unlock()
	return setting
}

// SetCheckRedirect specifies the policy for handling redirects.
//
// If checkRedirect is nil, the Client uses its default policy,
// which is to stop after 10 consecutive requests.
func (setting *HttpSettings) SetCheckRedirect(redirect func(req *http.Request, via []*http.Request) error) *HttpSettings {
	setting.mutex.Lock()
	setting.checkRedirect = redirect
	if setting.client != nil {
		setting.client.CheckRedirect = redirect
	}

	setting.mutex.Unlock()
	return setting
}

// DumpBody setting whether need to Dump the Body.
func (setting *HttpSettings) DumpBody(isdump bool) *HttpSettings {
	setting.mutex.Lock()
	setting.dumpBody = isdump
	setting.mutex.Unlock()
	return setting
}

// SetProxy set the http proxy
// example:
//
//	func(req *http.Request) (*url.URL, error) {
//		u, _ := url.ParseRequestURI("http://127.0.0.1:8118")
//		return u, nil
//	}
func (setting *HttpSettings) SetProxy(proxy func(*http.Request) (*url.URL, error)) *HttpSettings {
	setting.mutex.Lock()
	setting.proxy = proxy
	setting.mutex.Unlock()
	return setting
}

// SetDefaultSetting Overwrite default settings
func SetDefaultSetting(setting HttpSettings) {
	defaultSettingMutex.Lock()
	defaultSetting = setting
	defaultSettingMutex.Unlock()
}
func getDefaultSetting() (setting HttpSettings) {
	defaultSettingMutex.RLock()
	setting = defaultSetting
	defaultSettingMutex.RUnlock()
	return
}

func newDial(timeout time.Duration) *net.Dialer {
	return &net.Dialer{
		Timeout:   timeout,
		KeepAlive: 7 * time.Second, // 阿里云slb 如果15s没有发送任何请求会出现connectin reset 等错误
		DualStack: true,
	}

}
func setTransportTimeout(t *http.Transport, timeout time.Duration) {
	t.DialContext = newDial(timeout).DialContext
	t.ResponseHeaderTimeout = timeout
}
func getTransport(timeout time.Duration) *http.Transport {
	transport := &http.Transport{ // changed from http.DefaultTransport
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           newDial(timeout).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       7 * time.Second,
		MaxIdleConnsPerHost:   20,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     false,
		ResponseHeaderTimeout: timeout,
	}

	return transport
}

package httpclient

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptrace"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	gocontext "context"

	"github.com/aichy126/igo/log"
	"github.com/aichy126/igo/util"
)

// 不要重用 HttpRequest 对像，也不要用任何变量永久存储 HttpRequest 对象
// hTTPrequest设计之初就是goroutine不安全的，
// 一次请求 对应一个  HttpRequest对象

// 如果你想重用，请重用HttpSettings
// setting:=NewHttpSettings().SetDefaultTimeout(connectTimeout,timeout).SetUserAgent("ddns")
// httpRequest:=setting.Request(url,"GET")

// HttpSettings创建的一个 HttpRequest 表示一个http请求， 不要用同一个HttpRequest进行多次请求
// Get returns *HttpRequest with GET method.

// HttpRequest provides more useful methods for requesting one url than http.Request.
type HttpRequest struct {
	setting  HttpSettings // 不要改成指针，以避免setting的并发修改导致request请求异常
	method   string
	url      string
	body     io.Reader
	header   http.Header
	params   map[string][]string
	files    map[string]string
	username string
	password string
	host     string // req.Host = host
	version  string
	timeout  time.Duration // 此request的timeout,使用context.Withtimeout实现，不是在transport 级实现的
	retries  int           // 重试次数，默认0 不重试 if set to -1 means will retry forever
	ctx      gocontext.Context
	err      error

	acceptGzip  bool // add Accept-Encoding:gzip in request header
	reqCallBack func(req *http.Request) error

	dump []byte
}

func (b *HttpRequest) reset() {
	b.method = ""
	b.url = ""
	b.header = nil
	if len(b.files) > 0 {
		b.files = nil
	}
	b.setting = getDefaultSetting()
	b.dump = nil
	b.retries = 0
	b.username = ""
	b.password = ""
	b.host = ""
	b.version = ""
	b.timeout = 0
	b.err = nil

}

// SetBasicAuth sets the request's Authorization header to use HTTP Basic Authentication with the provided username and password.
func (b *HttpRequest) SetBasicAuth(username, password string) *HttpRequest {
	b.username = username
	b.password = password
	return b
}

func (b *HttpRequest) HeaderNX(key, value string) *HttpRequest {
	if b.header == nil {
		b.header = make(http.Header)
	}
	if b.header.Get(key) == "" {
		b.header.Set(key, value)
	}
	return b
}

// Header add header item string in request.
func (b *HttpRequest) Header(key, value string) *HttpRequest {
	if b.header == nil {
		b.header = make(http.Header)
	}

	b.header.Set(key, value)
	return b
}
func (b *HttpRequest) Headers(h http.Header) *HttpRequest {
	if h != nil {
		if b.header == nil {
			b.header = h
		} else {
			for key, value := range h {
				b.header[key] = value
			}
		}
	}

	return b
}

func (b *HttpRequest) SetAcceptEncodingGzip() *HttpRequest {
	if b.header == nil {
		b.header = make(http.Header)
	}
	b.header.Set("Accept-Encoding", "gzip")
	return b
}

func (b *HttpRequest) SetAutoGunzip(v bool) *HttpRequest {
	b.setting.gzip = v
	return b
}

// SetHost set the request host
func (b *HttpRequest) SetHost(host string) *HttpRequest {
	b.host = host
	return b
}

// SetProtocolVersion Set the protocol version for incoming requests.
// Client requests always use HTTP/1.1.
func (b *HttpRequest) SetProtocolVersion(version string) *HttpRequest {
	if len(version) == 0 {
		version = "HTTP/1.1"
	}
	b.version = version

	return b
}

// request 级的timeout ,只会地当前request
// 每次请求前都需要设置才生效
func (b *HttpRequest) SetTimeout(timeout time.Duration) *HttpRequest {
	b.timeout = timeout
	return b
}

// Retries sets Retries times.
// default is 0 means no retried.
// others means retried times.
func (b *HttpRequest) SetRetries(times int) *HttpRequest {
	if times < 0 {
		return b
	}
	b.retries = times
	return b
}
func (b *HttpRequest) SetEnableCookie(enable bool) *HttpRequest {
	b.setting.SetEnableCookie(enable)
	return b
}
func (b *HttpRequest) SetTLSClientConfig(config *tls.Config) *HttpRequest {
	b.setting.SetTLSClientConfig(config)
	b.setting.checkClient()
	return b
}

func (b *HttpRequest) SetTransport(transport http.RoundTripper) *HttpRequest {
	b.setting.SetTransport(transport)
	return b
}
func (b *HttpRequest) SetProxy(proxy func(*http.Request) (*url.URL, error)) *HttpRequest {
	b.setting.SetProxy(proxy)
	b.setting.checkClient()
	return b
}
func (b *HttpRequest) SetCheckRedirect(redirect func(req *http.Request, via []*http.Request) error) *HttpRequest {
	b.setting.SetCheckRedirect(redirect)
	return b
}

// SetUserAgent sets User-Agent header field
func (b *HttpRequest) SetUserAgent(useragent string) *HttpRequest {
	b.setting.SetUserAgent(useragent)
	return b
}
func (b *HttpRequest) Debug(debug bool) *HttpRequest {
	b.setting.Debug(debug)
	return b
}
func (b *HttpRequest) DumpBody(isdump bool) *HttpRequest {
	b.setting.DumpBody(isdump)
	return b
}

// SetCookie add cookie into request.
func (b *HttpRequest) SetCookie(cookie *http.Cookie) *HttpRequest {
	if b.header == nil {
		b.header = make(http.Header)
	}
	b.header.Add("Cookie", cookie.String())
	// b.header.Add("Cookie", cookie.String())
	return b
}

func (b *HttpRequest) Params(kv map[string]string) *HttpRequest {
	for key, value := range kv {
		b.Param(key, value)
	}
	return b
}

// Param adds query param in to request.
// params build query string as ?key1=value1&key2=value2...
func (b *HttpRequest) Param(key, value string) *HttpRequest {
	if b.params == nil {
		b.params = map[string][]string{}
	}

	if param, ok := b.params[key]; ok {
		b.params[key] = append(param, value)
	} else {
		b.params[key] = []string{value}
	}
	return b
}

// PostFile add a post file to the request
func (b *HttpRequest) PostFile(formname, filename string) *HttpRequest {
	if b.files == nil {
		b.files = make(map[string]string)
	}

	b.files[formname] = filename
	return b
}

// Body adds request raw body.
// it supports string and []byte.
func (b *HttpRequest) Body(data interface{}) *HttpRequest {
	switch t := data.(type) {
	case string:
		b.body = bytes.NewReader([]byte(t))
	case []byte:
		b.body = bytes.NewReader(t)
	case url.Values:
		b.body = bytes.NewReader([]byte(t.Encode()))
		b.HeaderNX("Content-Type", "application/x-www-form-urlencoded")
	case io.Reader:
		// if _, ok := t.(io.ReadSeeker); !ok {
		// 	log.Error("the param of HttpRequest.Body(obj)  is not an io.ReadSeeker,so it doesnot support retry ")
		// }
		b.body = t
	case nil:
		b.body = nil
	default:
		inData, err := json.Marshal(t)
		if err != nil {
			log.Error("json.Marshal", log.String("err", err.Error()))
			b.err = err
			return b
		}
		b.body = bytes.NewReader(inData)
		b.HeaderNX("Content-Type", "application/xml; charset=utf-8")

	}
	return b
}

// XMLBody adds request raw body encoding by XML.
func (b *HttpRequest) XMLBody(obj interface{}) *HttpRequest {
	b.HeaderNX("Content-Type", "application/xml; charset=utf-8")
	switch t := obj.(type) {
	case string:
		b.body = bytes.NewReader([]byte(t))
	case []byte:
		b.body = bytes.NewReader(t)
	case io.Reader: // 尽量是实现了ReadSeeker，的io.Readeer,否则不支持重试,因为一个Reader只能被读取一次
		b.body = t
		if _, ok := t.(io.ReadSeeker); !ok {
			log.Error("the param of HttpRequest.XMLBoDy(obj)  is not an io.ReadSeeker,so it doesnot support retry ")
		}

	case nil:
		b.body = nil
	default:
		byts, err := xml.Marshal(obj)
		if err != nil {
			log.Error("HttpRequest.XMLBody(data) xml.Marshal error", log.String("info", err.Error()))
			return b
		}
		b.body = bytes.NewReader(byts)
	}
	return b
}

// JSONBody adds request raw body encoding by JSON.
func (b *HttpRequest) JSONBody(obj interface{}) *HttpRequest {
	b.HeaderNX("Content-Type", "application/json; charset=utf-8")
	switch t := obj.(type) {
	case string:

		b.body = bytes.NewReader([]byte(t))
	case []byte:
		b.body = bytes.NewReader(t)
	case io.Reader: // 尽量是实现了ReadSeeker，的io.Readeer,否则不支持重试,因为一个Reader只能被读取一次
		b.body = t
		if _, ok := t.(io.ReadSeeker); !ok {
			log.Error("the param of HttpRequest.JSONBody(obj)  is not an io.ReadSeeker,so it doesnot support retry ")
		}
	case nil:
		b.body = nil
	default:
		inData, err := json.Marshal(t)
		if err != nil {
			log.Error("HttpRequest.JSONBody(data) json.Marshal error", log.String("info", err.Error()))
			b.err = err
			return b
		}
		b.body = bytes.NewReader(inData)
	}

	return b
}

func (b *HttpRequest) buildURL(paramBody string) {
	// build GET url with query string
	if b.method == "GET" && len(paramBody) > 0 {
		if strings.Contains(b.url, "?") {
			b.url += "&" + paramBody
		} else {
			b.url = b.url + "?" + paramBody
		}
		return
	}
	if b.method == "GET" && len(b.files) > 0 {
		b.method = "POST"
	}

	// build POST/PUT/PATCH url and body
	if (b.method == "POST" || b.method == "PUT" || b.method == "PATCH" || b.method == "DELETE") && b.body == nil {
		// with files
		if len(b.files) > 0 {
			pr, pw := io.Pipe()
			bodyWriter := multipart.NewWriter(pw)
			util.GoroutineFunc(func() {
				for formname, filename := range b.files {
					fileWriter, err := bodyWriter.CreateFormFile(formname, filename)
					if err != nil {
						// b.log.ContextErrorf(ctx,"httpclient.http_request.go:%v", err)
						b.err = err
					}
					fh, err := os.Open(filename)
					if err != nil {
						b.err = err
						// b.log.ContextErrorf(ctx,"httpclient.http_request.go:%v", err)
					}
					//iocopy
					_, err = io.Copy(fileWriter, fh)
					fh.Close()
					if err != nil {
						b.err = err
						// b.log.ContextErrorf(ctx,"httpclient.http_request.go:%v", err)
					}
				}
				for k, v := range b.params {
					for _, vv := range v {
						bodyWriter.WriteField(k, vv)
					}
				}
				bodyWriter.Close()
				pw.Close()
			})
			b.HeaderNX("Content-Type", bodyWriter.FormDataContentType())
			b.body = pr
			return
		}

		// with params
		if len(paramBody) > 0 {
			b.HeaderNX("Content-Type", "application/x-www-form-urlencoded")
			b.Body(paramBody)
		}
	}
}

// 注意此函数需要在你填充了所有param,header之后，用于获取组装好的Request,在你调用完此函数之后
// 你对 b.Param() b.Header() b.Body()等的调用将不再有效，因为http.Request在你调用GetRequest()的那一刻已生成
// 当然你可以直接对所回的req 任何修改。因为b里也何存的返回的req对象，
// 所以通过b.Bytes(),b.ToJSON()会使用你修改后的req对象进行请求
//
// httpclientReq:= b.Param("a","aaa").Header("k","v").Body("hello")
// req,err:=httpclientReq.GetRequest()
// req.Header.Set("new_key", "new_value")
// httpclientReq.ToJSON(ctx,&result)
//
// func (b *HttpRequest) GetRequest(ctx gocontext.Context) (req *http.Request, err error) {
// 	return b.generateRequest(ctx)
// }

// 在真正发起请求之前，允许你对Request对象进行修改
func (b *HttpRequest) SetRequestHook(f func(req *http.Request) error) *HttpRequest {
	b.reqCallBack = f
	return b
}

func (b *HttpRequest) do(ctx gocontext.Context, req *http.Request, retryCount int) (resp *http.Response, cancel gocontext.CancelFunc, err error) {
	if ctx == nil {
		ctx = gocontext.Background()
	}

	//try to start child span

	if b.setting.debug {
		dump, err := httputil.DumpRequest(req, b.setting.dumpBody)
		if err != nil {
			log.Error("httpclient.http_request.go", log.String("info", err.Error()))
		}
		b.dump = dump
	}

	var dnsStart, connStart, reqStart, waitResStart time.Duration
	var dnsDuration, connDuration, reqDuration, waitResDuration, finishDuration time.Duration
	// retries default value is 0, it will run once.
	// retries equal to -1, it will run forever until success
	// retries is setted, it will retries fixed times.
	var originCtx = req.Context()
	var reuseConn bool
	connStart, dnsStart, dnsDuration, connDuration, reqDuration, waitResDuration = -1, -1, -1, -1, -1, -1
	reuseConn = false
	startTime := time.Now()
	trace := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			dnsStart = time.Since(startTime)
			//log.Info("dns_start_callback", log.Any("info", info.Host))
		},
		DNSDone: func(dnsInfo httptrace.DNSDoneInfo) {
			dnsDuration = time.Since(startTime) - dnsStart
			//	log.Error("dns_done_callback", log.Any("addr", dnsInfo.Addrs), log.Any("err", dnsInfo.Err))
		},
		GetConn: func(h string) {
			connStart = time.Since(startTime)
			//log.Info("getconn_callback", log.Any("info", h))
		},
		GotConn: func(connInfo httptrace.GotConnInfo) {
			//log.Info("gotconn_callback", log.Any("reused", connInfo.Reused), log.Any("localaddr", connInfo.Conn.LocalAddr().String()), log.Any("remoteaddr", connInfo.Conn.RemoteAddr().String()))
			if connInfo.Reused {
				reuseConn = true
				connDuration = 0
			} else {
				connDuration = time.Since(startTime) - connStart
				reuseConn = false
			}
			reqStart = time.Since(startTime)
		},
		WroteRequest: func(w httptrace.WroteRequestInfo) {
			reqDuration = time.Since(startTime) - reqStart
			waitResStart = time.Since(startTime)
			//	log.Info("write_request_callback", log.Any("err", w.Err))
		},
		GotFirstResponseByte: func() {
			waitResDuration = time.Since(startTime) - waitResStart
		},
	}
	httpCtx := originCtx
	var timeout = b.timeout
	if timeout == 0 {
		timeout = b.setting.timeout
	}

	if timeout > 0 {
		httpCtx, cancel = gocontext.WithTimeout(httpCtx, timeout)
		// https://groups.google.com/forum/#!topic/golang-nuts/2FKwG6oEvos
		// 不可以直接在这里调 defer cancel()
		// cancel 要在 response.Body被读后调，否则 会收到 gocontext canceled
	} else {
		cancel = doNothing
	}

	req = req.WithContext(httptrace.WithClientTrace(httpCtx, trace))
	resp, err = b.setting.client.Do(req)
	finishDuration = time.Since(startTime)
	if err != nil {
		if dnsStart == -1 { //  //对于ip直连的，根本不需要dns
			dnsDuration = 0
		}

		if dnsStart != -1 && dnsDuration == -1 && connStart == -1 { //connStart!=-1时，说明已经进入conn阶段，此种情况有可能是reuse connection,不需要dns
			dnsDuration = finishDuration - dnsStart
		} else if connDuration == -1 {
			connDuration = finishDuration - dnsDuration
		} else if reqDuration == -1 {
			reqDuration = finishDuration - dnsDuration - connDuration
		} else if waitResDuration == -1 {
			waitResDuration = finishDuration - dnsDuration - connDuration - reqDuration
		}

		statInfo := fmt.Sprintf("retry:%d,total_cost:%s,dns_cost:%s,conn_cost:%s,reuse_conn:%v,req_cost:%s,wait_cost:%s",
			retryCount, formatDuration(finishDuration), formatDuration(dnsDuration),
			formatDuration(connDuration), reuseConn, formatDuration(reqDuration), formatDuration(waitResDuration))
		select {
		case <-req.Context().Done():
			err = errTimeout(err.Error()+" url:"+b.url+",stats="+statInfo, err)
			log.Error("errTimeout", log.String("info", err.Error()))
		default:
			err = errHttpClient(err.Error()+",stats="+statInfo, err)
			log.Error("errHttpClient", log.String("info", err.Error()))
		}
	} else {
		if b.setting.debug {
			log.Info("msg",
				log.String("url", b.url),
				log.Any("retry", retryCount),
				log.Any("total_cost", formatDuration(finishDuration)),
				log.Any("dns_cost", formatDuration(dnsDuration)),
				log.Any("conn_cost", formatDuration(connDuration)),
				log.Any("reuse_conn", reuseConn),
				log.Any("req_cost", formatDuration(reqDuration)),
				log.Any("wait_cost", formatDuration(waitResDuration)),
			)
		}
	}

	return

}

// Response will do the client.Do
func (b *HttpRequest) generateRequest(ctx gocontext.Context) (req *http.Request, err error) {
	if b.err != nil {
		return nil, err
	}

	if ctx == nil {
		ctx = gocontext.Background()
	}

	var paramBody string
	if len(b.params) > 0 {
		var buf bytes.Buffer
		for k, v := range b.params {
			for _, vv := range v {
				buf.WriteString(url.QueryEscape(k))
				buf.WriteByte('=')
				buf.WriteString(url.QueryEscape(vv))
				buf.WriteByte('&')
			}
		}
		paramBody = buf.String()
		paramBody = paramBody[0 : len(paramBody)-1]
	}
	if b.method == "" {
		return nil, errors.New("Method is empty")
	}

	b.buildURL(paramBody)
	if b.body != nil {
		if _, ok := b.body.(io.ReadSeeker); !ok {
			b.retries = 0
		}
	}

	req, err = newRequest(ctx, b.method, b.url, b.body, b.header)
	if err != nil {
		return nil, err
	}
	if b.username != "" {
		req.SetBasicAuth(b.username, b.password)
	}
	if b.version != "" {
		major, minor, ok := http.ParseHTTPVersion(b.version)
		if ok {
			req.Proto = b.version
			req.ProtoMajor = major
			req.ProtoMinor = minor
		}
	}
	if b.host != "" {
		req.Host = b.host
	}

	if b.setting.userAgent != "" && req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", b.setting.userAgent)
	}

	return req, err
}
func (b *HttpRequest) doRequestOnce(ctx gocontext.Context, retryCount int) (resp *http.Response, cancel gocontext.CancelFunc, err error) {
	req, err := b.generateRequest(ctx)
	if err != nil {
		return nil, doNothing, err
	}
	if req != nil && b.reqCallBack != nil {
		err = b.reqCallBack(req)
		if err != nil {
			return nil, doNothing, err
		}
	}
	resp, cancel, err = b.do(ctx, req, retryCount)
	if err == nil {
		return resp, cancel, nil
	}
	b.err = err
	// 为下次重试reset body
	if b.body != nil {
		if rs, ok := b.body.(io.ReadSeeker); ok {
			_, seekErr := rs.Seek(0, io.SeekStart)
			if seekErr != nil {
				return nil, doNothing, seekErr
			}
		} else {
			return nil, doNothing, errUnseekable
		}
	}
	return resp, cancel, err

}
func (b *HttpRequest) doRequest(ctx gocontext.Context) (resp *http.Response, cancel gocontext.CancelFunc, err error) {
	var retryCount int
	for ; b.retries == -1 || retryCount <= b.retries; retryCount++ {
		resp, cancel, err = b.doRequestOnce(ctx, retryCount)
		if err == nil {
			return resp, cancel, err
		}
		if err == errUnseekable {
			log.Error("HttpRequest.body is an io.Reader and not an io.ReadSeeker so it does not support retry")
			return nil, doNothing, b.err
		}

		// 上一次的请求若产生了err,下一次请求的时候会忽略此err
		b.err = nil
	}
	if err == nil {
		return
	}
	// connection reset by peer通常是keepalive的连接对端已经关了， 故对此种类型的error主动retry一次
	if strings.Contains(err.Error(), "connection reset by peer") || strings.Contains(err.Error(), "EOF") ||
		strings.Contains(err.Error(), "http: server closed idle connection") {
		log.Error("connection_r_eset_by_peer", log.Any("retryCount", retryCount+1))
		return b.doRequestOnce(ctx, retryCount+1)
	}

	return
}

// Response executes request client gets response mannually.
func (b *HttpRequest) Response(ctx gocontext.Context) (*http.Response, error) {
	response, cancel, err := b.doRequest(ctx)
	defer cancel()

	if err != nil {
		return response, err
	}
	err = readResponseBody(response)

	return response, err
}

// String returns the body string in response.
// it calls Response inner.
func (b *HttpRequest) String(ctx gocontext.Context) (string, error) {
	data, err := b.Bytes(ctx)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// BytesWithStatus returns the StatusCode in response, and body []byte in response.
// it calls Response inner.
func (b *HttpRequest) BytesWithStatus(ctx gocontext.Context) (int, []byte, error) {
	resp, err := b.Response(ctx)
	if resp != nil {
		defer resp.Body.Close()
	}
	if resp == nil {
		return http.StatusInternalServerError, nil, err
	}

	if err != nil {
		return resp.StatusCode, nil, err
	}
	if resp.Body == nil {
		return resp.StatusCode, nil, nil
	}

	if b.setting.gzip && strings.ToLower(resp.Header.Get("Content-Encoding")) == "gzip" {
		reader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return http.StatusBadRequest, nil, err
		}
		respBody, err := ioutil.ReadAll(reader)
		return resp.StatusCode, respBody, err
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	return resp.StatusCode, respBody, err

}

// Bytes returns the body []byte in response.
// it calls Response inner.
func (b *HttpRequest) Bytes(ctx gocontext.Context) ([]byte, error) {
	_, body, err := b.BytesWithStatus(ctx)
	return body, err
}

// ToFile saves the body data in response to one file.
// it calls Response inner.
func (b *HttpRequest) ToFile(ctx gocontext.Context, filename string) error {
	f, err := os.Create(filename)
	// fn, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
	// if err != nil {
	// 	return fmt.Errorf("Open file %s fail:%s", filename, err.Error())
	// }
	if err != nil {
		return err
	}
	defer f.Close()

	resp, err := b.Response(ctx)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	}
	if resp.Body == nil {
		return nil
	}
	_, err = io.Copy(f, resp.Body)
	return err
}

// ToJSON returns the map that marshals from the body bytes as json in response .
// it calls Response inner.
func (b *HttpRequest) ToJSON(ctx gocontext.Context, v interface{}) error {
	data, err := b.Bytes(ctx)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, v)
	if err == nil {
		return nil
	}

	return errors.New(err.Error() + ",content:" + string(data))
}
func (b *HttpRequest) ToJSONWhenStatusOk(ctx gocontext.Context, v interface{}) (code int, err error) {
	code, data, err := b.BytesWithStatus(ctx)
	if err != nil {
		return code, err
	}
	if code == http.StatusOK {
		err = json.Unmarshal(data, v)
	}

	if err != nil {
		return code, errors.New(err.Error() + ",content:" + string(data))
	}
	return code, err
}
func (b *HttpRequest) ToJSONWithStatus(ctx gocontext.Context, v interface{}) (code int, err error) {
	code, data, err := b.BytesWithStatus(ctx)
	if err != nil {
		return code, err
	}
	err = json.Unmarshal(data, v)
	if err != nil {
		return code, errors.New(err.Error() + ",content:" + string(data))
	}

	return code, err
}

// ToXML returns the map that marshals from the body bytes as xml in response .
// it calls Response inner.
func (b *HttpRequest) ToXML(ctx gocontext.Context, v interface{}) error {
	data, err := b.Bytes(ctx)
	if err != nil {
		return err
	}
	return xml.Unmarshal(data, v)
}

func (b *HttpRequest) IgnoreResponse(ctx gocontext.Context) error {
	res, err := b.Response(ctx)
	if err != nil {
		return err
	}
	if res != nil {
		res.Body.Close()
	}
	return nil
}

// DumpRequest return the DumpRequest
func (b *HttpRequest) GetDumpRequest() []byte {
	return b.dump
}
func (b *HttpRequest) GetContext() gocontext.Context {
	return b.ctx
}
func (b *HttpRequest) Error() error {
	return b.err
}

func doNothing() {}

type httpError struct {
	errMsg  string
	timeout bool
	err     error
}

func (e *httpError) Error() string   { return e.errMsg }
func (e *httpError) Timeout() bool   { return e.timeout }
func (e *httpError) Temporary() bool { return true }

type timeourErr interface {
	Timeout() bool
}

func errTimeout(msg string, err error) error {
	return &httpError{errMsg: "httpclient/http_request.go: timeout " + msg, timeout: true, err: err}
}
func errHttpClient(msg string, err error) error {
	return &httpError{errMsg: "httpclient/http_request.go: " + msg, timeout: false, err: err}
}

var errUnseekable = errors.New("request.body is not seekable,unsupport http retry")

func IsTimeout(err error) bool {
	if err == nil {
		return false
	}

	if t, ok := err.(timeourErr); ok {
		return t.Timeout()
	}
	return false
}
func formatDuration(t time.Duration) string {
	if t == -1 {
		return "?"
	}
	if t == 0 {
		return "0"
	}

	i := (t / time.Millisecond).Nanoseconds()
	if i > 0 {
		return fmt.Sprintf("%dms", i)
	}

	m := (t / time.Microsecond).Nanoseconds()
	if m == 0 {
		m = 1
	}

	return fmt.Sprintf("%dµ", m)
	// ctx.VDepth(10, 1).Info("nanasecond=", t)
	//
}

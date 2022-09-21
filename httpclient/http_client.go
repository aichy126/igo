package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	gocontext "context"

	"github.com/aichy126/igo/context"
)

// HttpClient只是对HttpSettings的再次封装，实际HttpSettings完全可以独立对外提供接口
// 详见http_setting.go开头的示例
// HttpClient直接继承自HttpSettings，故可以直接用它提供的接口

// 用法 client:=NewClient().Debug(true).SetDefaultTimeout(timeout)
var DefaultClient = NewClient()

type HttpClient = HttpSettings

type HttpClientOption func(*HttpClient)

func WithTimeoutOpt(timeout time.Duration) HttpClientOption {
	return func(hCli *HttpClient) {
		hCli.SetDefaultTimeout(timeout)
	}
}

func NewClient(opts ...HttpClientOption) *HttpClient {
	timeout := 2 * time.Second
	hCli := NewHttpSettings().Debug(false).DumpBody(false).
		SetDefaultTimeout(timeout)
	for _, opt := range opts {
		opt(hCli)
	}
	return hCli
}

func (c *HttpClient) Do(ctx context.IContext, req *http.Request) (*http.Response, error) {
	r, cancel, err := c.ContextRequest(ctx, req.URL.RawPath, req.Method).do(ctx, req, 0)
	defer cancel()
	if err != nil {
		return r, err
	}
	// https://groups.google.com/forum/#!topic/golang-nuts/2FKwG6oEvos
	// cancel 要在 response.Body被读后调，否则 会收到 context canceled
	// 故我们readResponseBody 读取response.body的内容
	err = readResponseBody(r)
	return r, err
}

// warning 需要你来将resp.body close
func (c *HttpClient) Get(ctx context.IContext, url string) (resp *http.Response, err error) {
	r, err := c.ContextRequestGet(ctx, url).Response(ctx)
	if err != nil {
		return r, err
	}
	// https://groups.google.com/forum/#!topic/golang-nuts/2FKwG6oEvos
	// cancel 要在 response.Body被读后调，否则 会收到 context canceled
	// 故我们readResponseBody 读取response.body的内容
	err = readResponseBody(r)
	return r, err
}

// warning 需要你来将resp.body close
func (c *HttpClient) Post(ctx context.IContext, url string, contentType string, body io.Reader) (resp *http.Response, err error) {
	r, err := c.ContextRequestPost(ctx, url).
		Header("Content-Type", contentType).Body(body).
		Response(ctx)
	if err != nil {
		return r, err
	}
	// https://groups.google.com/forum/#!topic/golang-nuts/2FKwG6oEvos
	// cancel 要在 response.Body被读后调，否则 会收到 context canceled
	// 故我们readResponseBody 读取response.body的内容
	err = readResponseBody(r)
	return r, err
}
func (c *HttpClient) Put(ctx context.IContext, url string, contentType string, body io.Reader) (resp *http.Response, err error) {
	r, err := c.ContextRequestPut(ctx, url).
		Header("Content-Type", contentType).Body(body).
		Response(ctx)
	if err != nil {
		return r, err
	}
	// https://groups.google.com/forum/#!topic/golang-nuts/2FKwG6oEvos
	// cancel 要在 response.Body被读后调，否则 会收到 context canceled
	// 故我们readResponseBody 读取response.body的内容
	err = readResponseBody(r)
	return r, err
}
func (c *HttpClient) PostBytes(ctx context.IContext, url string, contentType string, body interface{}) (bytes []byte, err error) {
	return c.ContextRequestPost(ctx, url).
		Header("Content-Type", contentType).Body(body).Bytes(ctx)

}

// warning 需要你来将resp.body close
func (c *HttpClient) PostForm(ctx context.IContext, url string, data url.Values) (resp *http.Response, err error) {
	encodedData := data.Encode()
	return c.Post(ctx, url, "application/x-www-form-urlencoded", strings.NewReader(encodedData))
}

func (c *HttpClient) GetJsonWithHeader(ctx context.IContext, urlStr string, header http.Header) (data []byte, err error) {
	data, err = c.RequestJsonWithHeader(ctx, urlStr, "GET", header, nil)
	return
}

func (c *HttpClient) GetJsonWithHeaderAs(ctx context.IContext, urlStr string, header http.Header, target interface{}) (err error) {
	data, err := c.RequestJsonWithHeader(ctx, urlStr, "GET", header, nil)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("can't unmarshal to target %s %s", err, string(data))
	}
	return
}

func (c *HttpClient) PostJsonWithHeaderAs(ctx context.IContext, urlStr string, header http.Header, content interface{}, target interface{}) (err error) {
	data, err := c.RequestJsonWithHeader(ctx, urlStr, "POST", header, content)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("can't unmarshal to target %s %s", err, string(data))
	}
	return
}

func (c *HttpClient) PostFormWithHeaderAs(ctx context.IContext, urlStr string, header http.Header, v url.Values, target interface{}) (err error) {
	data, err := c.requestWithHeader(ctx, urlStr, "POST", "application/x-www-form-urlencoded", header, v)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("can't unmarshal to target %s %s", err, string(data))
	}
	return nil

}

func (c *HttpClient) PostFormFileWithHeaderAs(ctx context.IContext, urlStr string, contentType string, header http.Header, v interface{}, target interface{}) (err error) {
	data, err := c.requestWithHeader(ctx, urlStr, "POST", contentType, header, v)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("can't unmarshal to target %s %s", err, string(data))
	}
	return nil

}

func (c *HttpClient) PostFormWithHeader(ctx context.IContext, urlStr string, header http.Header, v url.Values) (data []byte, err error) {
	return c.requestWithHeader(ctx, urlStr, "POST", "application/x-www-form-urlencoded", header, v)
}

func (c *HttpClient) RequestJsonWithHeader(ctx context.IContext, urlStr string, method string, header http.Header, content interface{}) (data []byte, err error) {
	return c.requestWithHeader(ctx, urlStr, method, "application/json", header, content)

}

func (c *HttpClient) requestWithHeader(ctx context.IContext, urlStr string, method string, contentType string, header http.Header, content interface{}) (data []byte, err error) {
	if header == nil {
		header = make(http.Header)
	}

	return c.Request(urlStr, method).Headers(header).
		Body(content).Header("Content-Type", contentType).Bytes(ctx)
}

func (c *HttpClient) GetWithHeaderAs(ctx context.IContext, urlStr string, header http.Header, target interface{}) (err error) {
	data, err := c.requestWithHeader(ctx, urlStr, "GET", "application/x-www-form-urlencoded", header, nil)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("can't unmarshal to target %s %s", err, string(data))
	}

	return
}

func (c *HttpClient) GetWithHeader(ctx context.IContext, urlStr string, header http.Header) (data []byte, err error) {
	data, err = c.requestWithHeader(ctx, urlStr, "GET", "application/x-www-form-urlencoded", header, nil)
	return
}

func (c *HttpClient) GetBytes(ctx context.IContext, url string) ([]byte, error) {
	return c.ContextRequestGet(ctx, url).Bytes(ctx)
}
func (c *HttpClient) GetBytesWithStatus(ctx context.IContext, url string) (int, []byte, error) {
	return c.ContextRequestGet(ctx, url).BytesWithStatus(ctx)
}

// 从url获取返回,并以json格式解析到target
func (c *HttpClient) GetAs(ctx context.IContext, url string, target interface{}) error {
	return c.ContextRequestGet(ctx, url).ToJSON(ctx, target)
}

func (c *HttpClient) GetAsWhenStatusOk(ctx context.IContext, url string, target interface{}) (int, error) {
	return c.ContextRequestGet(ctx, url).ToJSONWhenStatusOk(ctx, target)
}

func (c *HttpClient) PostFormAs(ctx context.IContext, url string, v url.Values, target interface{}) error {
	return c.ContextRequestPost(ctx, url).Body(v).ToJSON(ctx, target)
}

func (c *HttpClient) PostFormBytes(ctx context.IContext, urlStr string, v url.Values) (body []byte, err error) {
	return c.ContextRequestPost(ctx, urlStr).
		Body(v).Bytes(ctx)
}
func (c *HttpClient) PostFormBytesWithStatus(ctx context.IContext, urlStr string, v url.Values) (code int, body []byte, err error) {
	return c.ContextRequestPost(ctx, urlStr).
		Body(v).BytesWithStatus(ctx)
}
func (c *HttpClient) PostJsonBytesWithStatus(ctx context.IContext, urlStr string, content interface{}) (code int, body []byte, err error) {
	return c.ContextRequestPost(ctx, urlStr).JSONBody(content).BytesWithStatus(ctx)
}

func (c *HttpClient) PostJsonAs(ctx context.IContext, urlStr string, content interface{}, target interface{}) (err error) {
	return c.ContextRequestPost(ctx, urlStr).JSONBody(content).ToJSON(ctx, target)
}

func (c *HttpClient) PutJsonAs(ctx context.IContext, urlStr string, content interface{}, target interface{}) (err error) {
	return c.ContextRequestPut(ctx, urlStr).JSONBody(content).ToJSON(ctx, target)
}

func (c *HttpClient) PutAsJson(ctx context.IContext, urlStr string, content interface{}) (data []byte, err error) {
	return c.ContextRequestPut(ctx, urlStr).JSONBody(content).Bytes(ctx)
}

func (c *HttpClient) PostJson(ctx context.IContext, urlStr string, content interface{}) (data []byte, err error) {
	return c.ContextRequestPost(ctx, urlStr).JSONBody(content).Bytes(ctx)
}

// warning 需要你来将resp.body close
func Get(ctx context.IContext, url string) (resp *http.Response, err error) {
	return DefaultClient.Get(ctx, url)
}

func NewRequest(ctx context.IContext, method, url string, body io.Reader) (*http.Request, error) {
	return newRequest(ctx, method, url, body, nil)
}
func newRequest(ctx gocontext.Context, method, url string, body io.Reader, header http.Header) (*http.Request, error) {
	var req *http.Request
	var err error
	if ctx == nil {
		ctx = context.NewContext()
	}

	req, err = http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if header != nil {
		req.Header = header
	}

	return req, nil

}

func readResponseBody(res *http.Response) error {
	if res == nil {
		return nil
	}
	defer res.Body.Close()
	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	res.Body = ioutil.NopCloser(bytes.NewReader(buf))
	return nil
}

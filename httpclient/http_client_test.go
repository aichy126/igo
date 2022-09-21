package httpclient

import (
	"testing"
	"time"

	"github.com/aichy126/igo/context"
	"github.com/davecgh/go-spew/spew"
)

func TestGet(t *testing.T) {
	timeout := time.Second * 30
	client := NewClient().Debug(false).SetDefaultTimeout(timeout).SetUserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/79.0.3945.88 Safari/537.36").SetDefaultRetries(3)
	ctx := context.Background()
	body, err := client.GetBytes(ctx, "https://autumnfish.cn/api/joke/list?num=1")
	spew.Dump(string(body), err)
}

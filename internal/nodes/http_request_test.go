package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/iwind/TeaGo/assert"
	"runtime"
	"testing"
)

func TestHTTPRequest_RedirectToHTTPS(t *testing.T) {
	var a = assert.NewAssertion(t)
	{
		req := &HTTPRequest{
			ReqServer: &serverconfigs.ServerConfig{
				Web: &serverconfigs.HTTPWebConfig{
					RedirectToHttps: &serverconfigs.HTTPRedirectToHTTPSConfig{},
				},
			},
		}
		req.Do()
		a.IsBool(req.web.RedirectToHttps.IsOn == false)
	}
	{
		req := &HTTPRequest{
			ReqServer: &serverconfigs.ServerConfig{
				Web: &serverconfigs.HTTPWebConfig{
					RedirectToHttps: &serverconfigs.HTTPRedirectToHTTPSConfig{
						IsOn: true,
					},
				},
			},
		}
		req.Do()
		a.IsBool(req.web.RedirectToHttps.IsOn == true)
	}
}

func TestHTTPRequest_Memory(t *testing.T) {
	var stat1 = &runtime.MemStats{}
	runtime.ReadMemStats(stat1)

	var requests = []*HTTPRequest{}
	for i := 0; i < 1_000_000; i++ {
		requests = append(requests, &HTTPRequest{})
	}

	var stat2 = &runtime.MemStats{}
	runtime.ReadMemStats(stat2)
	t.Log((stat2.HeapInuse-stat1.HeapInuse)/1024/1024, "MB,")
	t.Log(len(requests))
}

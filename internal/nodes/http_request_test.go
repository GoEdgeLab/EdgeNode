package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/iwind/TeaGo/assert"
	"testing"
)

func TestHTTPRequest_RedirectToHTTPS(t *testing.T) {
	a := assert.NewAssertion(t)
	{
		req := &HTTPRequest{
			Server: &serverconfigs.ServerConfig{
				Web: &serverconfigs.HTTPWebConfig{
					RedirectToHttps: &serverconfigs.HTTPRedirectToHTTPSConfig{},
				},
			},
		}
		req.Run()
		a.IsBool(req.web.RedirectToHttps.IsOn == false)
	}
	{
		req := &HTTPRequest{
			Server: &serverconfigs.ServerConfig{
				Web: &serverconfigs.HTTPWebConfig{
					RedirectToHttps: &serverconfigs.HTTPRedirectToHTTPSConfig{
						IsOn: true,
					},
				},
			},
		}
		req.Run()
		a.IsBool(req.web.RedirectToHttps.IsOn == true)
	}
}

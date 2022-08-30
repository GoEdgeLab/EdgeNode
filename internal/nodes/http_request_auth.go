// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"bytes"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"io"
	"net/http"
)

// 执行认证
func (this *HTTPRequest) doAuth() (shouldStop bool) {
	if this.web.Auth == nil || !this.web.Auth.IsOn {
		return
	}

	for _, ref := range this.web.Auth.PolicyRefs {
		if !ref.IsOn || ref.AuthPolicy == nil || !ref.AuthPolicy.IsOn {
			continue
		}
		if !ref.AuthPolicy.MatchRequest(this.RawReq) {
			continue
		}
		ok, newURI, uriChanged, err := ref.AuthPolicy.Filter(this.RawReq, func(subReq *http.Request) (status int, err error) {
			subReq.TLS = this.RawReq.TLS
			subReq.RemoteAddr = this.RawReq.RemoteAddr
			subReq.Host = this.RawReq.Host
			subReq.Proto = this.RawReq.Proto
			subReq.ProtoMinor = this.RawReq.ProtoMinor
			subReq.ProtoMajor = this.RawReq.ProtoMajor
			subReq.Body = io.NopCloser(bytes.NewReader([]byte{}))
			subReq.Header.Set("Referer", this.URL())
			var writer = NewEmptyResponseWriter(this.writer)
			this.doSubRequest(writer, subReq)
			return writer.StatusCode(), nil
		}, this.Format)
		if err != nil {
			this.write50x(err, http.StatusInternalServerError, "Failed to execute the AuthPolicy", "认证策略执行失败", false)
			return
		}
		if ok {
			if uriChanged {
				this.uri = newURI
			}
			this.tags = append(this.tags, ref.AuthPolicy.Type)
			return
		} else {
			// Basic Auth比较特殊
			if ref.AuthPolicy.Type == serverconfigs.HTTPAuthTypeBasicAuth {
				method, ok := ref.AuthPolicy.Method().(*serverconfigs.HTTPAuthBasicMethod)
				if ok {
					var headerValue = "Basic realm=\""
					if len(method.Realm) > 0 {
						headerValue += method.Realm
					} else {
						headerValue += this.ReqHost
					}
					headerValue += "\""
					if len(method.Charset) > 0 {
						headerValue += ", charset=\"" + method.Charset + "\""
					}
					this.writer.Header()["WWW-Authenticate"] = []string{headerValue}
				}
			}
			this.writer.WriteHeader(http.StatusUnauthorized)
			this.tags = append(this.tags, ref.AuthPolicy.Type)
			return true
		}
	}
	return
}

// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package waf

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"net/http"
)

type PageAction struct {
	BaseAction

	Status int    `yaml:"status" json:"status"`
	Body   string `yaml:"body" json:"body"`
}

func (this *PageAction) Init(waf *WAF) error {
	if this.Status <= 0 {
		this.Status = http.StatusForbidden
	}
	return nil
}

func (this *PageAction) Code() string {
	return ActionPage
}

func (this *PageAction) IsAttack() bool {
	return false
}

// WillChange determine if the action will change the request
func (this *PageAction) WillChange() bool {
	return true
}

// Perform the action
func (this *PageAction) Perform(waf *WAF, group *RuleGroup, set *RuleSet, request requests.Request, writer http.ResponseWriter) (continueRequest bool, goNextSet bool) {
	if writer == nil {
		return
	}

	request.ProcessResponseHeaders(writer.Header(), this.Status)
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(this.Status)

	var body = this.Body
	if len(body) == 0 {
		body = `<!DOCTYPE html>
<html lang="en">
<title>403 Forbidden</title>
	<style>
		address { line-height: 1.8; }
	</style>
<body>
<h1>403 Forbidden By WAF</h1>
<address>Connection: ${remoteAddr} (Client) -&gt; ${serverAddr} (Server)</address>
<address>Request ID: ${requestId}</address>
</body>
</html>`
	}
	_, _ = writer.Write([]byte(request.Format(body)))

	return false, false
}

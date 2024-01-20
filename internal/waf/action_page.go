// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package waf

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"net/http"
)

type PageAction struct {
	BaseAction

	UseDefault bool   `yaml:"useDefault" json:"useDefault"`
	Status     int    `yaml:"status" json:"status"`
	Body       string `yaml:"body" json:"body"`
}

func (this *PageAction) Init(waf *WAF) error {
	if waf.DefaultPageAction != nil {
		if this.Status <= 0 || this.UseDefault {
			this.Status = waf.DefaultPageAction.Status
		}
		if len(this.Body) == 0 || this.UseDefault {
			this.Body = waf.DefaultPageAction.Body
		}
	}

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
func (this *PageAction) Perform(waf *WAF, group *RuleGroup, set *RuleSet, request requests.Request, writer http.ResponseWriter) PerformResult {
	if writer == nil {
		return PerformResult{}
	}

	request.ProcessResponseHeaders(writer.Header(), this.Status)
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(this.Status)

	var body = this.Body
	if len(body) == 0 {
		body = `<!DOCTYPE html>
<html lang="en">
<head>
	<title>403 Forbidden</title>
	<style>
		address { line-height: 1.8; }
	</style>
</head>
<body>
<h1>403 Forbidden By WAF</h1>
<address>Connection: ${remoteAddr} (Client) -&gt; ${serverAddr} (Server)</address>
<address>Request ID: ${requestId}</address>
</body>
</html>`
	}
	_, _ = writer.Write([]byte(request.Format(body)))

	return PerformResult{}
}

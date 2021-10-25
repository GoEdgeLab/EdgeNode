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
func (this *PageAction) Perform(waf *WAF, group *RuleGroup, set *RuleSet, request requests.Request, writer http.ResponseWriter) (allow bool) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(this.Status)
	_, _ = writer.Write([]byte(request.Format(this.Body)))

	return false
}

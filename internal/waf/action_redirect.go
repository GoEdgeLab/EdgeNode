// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package waf

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"net/http"
)

type RedirectAction struct {
	BaseAction

	Status int    `yaml:"status" json:"status"`
	URL    string `yaml:"url" json:"url"`
}

func (this *RedirectAction) Init(waf *WAF) error {
	if this.Status <= 0 {
		this.Status = http.StatusTemporaryRedirect
	}
	return nil
}

func (this *RedirectAction) Code() string {
	return ActionRedirect
}

func (this *RedirectAction) IsAttack() bool {
	return false
}

// WillChange determine if the action will change the request
func (this *RedirectAction) WillChange() bool {
	return true
}

// Perform the action
func (this *RedirectAction) Perform(waf *WAF, group *RuleGroup, set *RuleSet, request requests.Request, writer http.ResponseWriter) (continueRequest bool, goNextSet bool) {
	request.ProcessResponseHeaders(writer.Header(), this.Status)
	writer.Header().Set("Location", this.URL)
	writer.WriteHeader(this.Status)

	return false, false
}

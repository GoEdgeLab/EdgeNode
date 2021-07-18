// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package waf

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"net/http"
)

type ActionInterface interface {
	// Init 初始化
	Init(waf *WAF) error

	// Code 代号
	Code() string

	// IsAttack 是否为拦截攻击动作
	IsAttack() bool

	// WillChange determine if the action will change the request
	WillChange() bool

	// Perform perform the action
	Perform(waf *WAF, group *RuleGroup, set *RuleSet, request requests.Request, writer http.ResponseWriter) (allow bool)
}

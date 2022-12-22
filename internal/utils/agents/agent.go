// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package agents

import (
	"regexp"
	"strings"
)

type Agent struct {
	Code     string
	Keywords []string // user agent keywords

	suffixes []string // PTR suffixes
	reg      *regexp.Regexp
}

func NewAgent(code string, suffixes []string, reg *regexp.Regexp, keywords []string) *Agent {
	return &Agent{
		Code:     code,
		suffixes: suffixes,
		reg:      reg,
		Keywords: keywords,
	}
}

func (this *Agent) Match(ptr string) bool {
	if len(this.suffixes) > 0 {
		for _, suffix := range this.suffixes {
			if strings.HasSuffix(ptr, suffix) {
				return true
			}
		}
	}
	if this.reg != nil {
		return this.reg.MatchString(ptr)
	}
	return false
}

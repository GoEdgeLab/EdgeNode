// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.
//go:build linux

package nftables

import (
	nft "github.com/google/nftables"
	"github.com/google/nftables/expr"
)

type Rule struct {
	rawRule *nft.Rule
}

func NewRule(rawRule *nft.Rule) *Rule {
	return &Rule{
		rawRule: rawRule,
	}
}

func (this *Rule) Raw() *nft.Rule {
	return this.rawRule
}

func (this *Rule) LookupSetName() string {
	for _, e := range this.rawRule.Exprs {
		exp, ok := e.(*expr.Lookup)
		if ok {
			return exp.SetName
		}
	}
	return ""
}

func (this *Rule) VerDict() expr.VerdictKind {
	for _, e := range this.rawRule.Exprs {
		exp, ok := e.(*expr.Verdict)
		if ok {
			return exp.Kind
		}
	}

	return -100
}

func (this *Rule) Handle() uint64 {
	return this.rawRule.Handle
}

func (this *Rule) UserData() []byte {
	return this.rawRule.UserData
}

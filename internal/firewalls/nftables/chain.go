// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.
//go:build linux

package nftables

import (
	"bytes"
	"errors"
	nft "github.com/google/nftables"
	"github.com/google/nftables/expr"
)

const MaxChainNameLength = 31

type RuleOptions struct {
	Exprs    []expr.Any
	UserData []byte
}

// Chain chain object in table
type Chain struct {
	conn     *Conn
	rawTable *nft.Table
	rawChain *nft.Chain
}

func NewChain(conn *Conn, rawTable *nft.Table, rawChain *nft.Chain) *Chain {
	return &Chain{
		conn:     conn,
		rawTable: rawTable,
		rawChain: rawChain,
	}
}

func (this *Chain) Raw() *nft.Chain {
	return this.rawChain
}

func (this *Chain) Name() string {
	return this.rawChain.Name
}

func (this *Chain) AddRule(options *RuleOptions) (*Rule, error) {
	var rawRule = this.conn.Raw().AddRule(&nft.Rule{
		Table:    this.rawTable,
		Chain:    this.rawChain,
		Exprs:    options.Exprs,
		UserData: options.UserData,
	})
	err := this.conn.Commit()
	if err != nil {
		return nil, err
	}
	return NewRule(rawRule), nil
}

func (this *Chain) AddAcceptIPv4Rule(ip []byte, userData []byte) (*Rule, error) {
	return this.AddRule(&RuleOptions{
		Exprs: []expr.Any{
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       12,
				Len:          4,
			},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     ip,
			},
			&expr.Verdict{
				Kind: expr.VerdictAccept,
			},
		},
		UserData: userData,
	})
}

func (this *Chain) AddAcceptIPv6Rule(ip []byte, userData []byte) (*Rule, error) {
	return this.AddRule(&RuleOptions{
		Exprs: []expr.Any{
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       8,
				Len:          16,
			},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     ip,
			},
			&expr.Verdict{
				Kind: expr.VerdictAccept,
			},
		},
		UserData: userData,
	})
}

func (this *Chain) AddDropIPv4Rule(ip []byte, userData []byte) (*Rule, error) {
	return this.AddRule(&RuleOptions{
		Exprs: []expr.Any{
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       12,
				Len:          4,
			},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     ip,
			},
			&expr.Verdict{
				Kind: expr.VerdictDrop,
			},
		},
		UserData: userData,
	})
}

func (this *Chain) AddDropIPv6Rule(ip []byte, userData []byte) (*Rule, error) {
	return this.AddRule(&RuleOptions{
		Exprs: []expr.Any{
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       8,
				Len:          16,
			},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     ip,
			},
			&expr.Verdict{
				Kind: expr.VerdictDrop,
			},
		},
		UserData: userData,
	})
}

func (this *Chain) AddRejectIPv4Rule(ip []byte, userData []byte) (*Rule, error) {
	return this.AddRule(&RuleOptions{
		Exprs: []expr.Any{
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       12,
				Len:          4,
			},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     ip,
			},
			&expr.Reject{},
		},
		UserData: userData,
	})
}

func (this *Chain) AddRejectIPv6Rule(ip []byte, userData []byte) (*Rule, error) {
	return this.AddRule(&RuleOptions{
		Exprs: []expr.Any{
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       8,
				Len:          16,
			},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     ip,
			},
			&expr.Reject{},
		},
		UserData: userData,
	})
}

func (this *Chain) AddAcceptIPv4SetRule(setName string, userData []byte) (*Rule, error) {
	return this.AddRule(&RuleOptions{
		Exprs: []expr.Any{
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       12,
				Len:          4,
			},
			&expr.Lookup{
				SourceRegister: 1,
				SetName:        setName,
			},
			&expr.Verdict{
				Kind: expr.VerdictAccept,
			},
		},
		UserData: userData,
	})
}

func (this *Chain) AddAcceptIPv6SetRule(setName string, userData []byte) (*Rule, error) {
	return this.AddRule(&RuleOptions{
		Exprs: []expr.Any{
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       8,
				Len:          16,
			},
			&expr.Lookup{
				SourceRegister: 1,
				SetName:        setName,
			},
			&expr.Verdict{
				Kind: expr.VerdictAccept,
			},
		},
		UserData: userData,
	})
}

func (this *Chain) AddDropIPv4SetRule(setName string, userData []byte) (*Rule, error) {
	return this.AddRule(&RuleOptions{
		Exprs: []expr.Any{
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       12,
				Len:          4,
			},
			&expr.Lookup{
				SourceRegister: 1,
				SetName:        setName,
			},
			&expr.Verdict{
				Kind: expr.VerdictDrop,
			},
		},
		UserData: userData,
	})
}

func (this *Chain) AddDropIPv6SetRule(setName string, userData []byte) (*Rule, error) {
	return this.AddRule(&RuleOptions{
		Exprs: []expr.Any{
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       8,
				Len:          16,
			},
			&expr.Lookup{
				SourceRegister: 1,
				SetName:        setName,
			},
			&expr.Verdict{
				Kind: expr.VerdictDrop,
			},
		},
		UserData: userData,
	})
}

func (this *Chain) AddRejectIPv4SetRule(setName string, userData []byte) (*Rule, error) {
	return this.AddRule(&RuleOptions{
		Exprs: []expr.Any{
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       12,
				Len:          4,
			},
			&expr.Lookup{
				SourceRegister: 1,
				SetName:        setName,
			},
			&expr.Reject{},
		},
		UserData: userData,
	})
}

func (this *Chain) AddRejectIPv6SetRule(setName string, userData []byte) (*Rule, error) {
	return this.AddRule(&RuleOptions{
		Exprs: []expr.Any{
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       8,
				Len:          16,
			},
			&expr.Lookup{
				SourceRegister: 1,
				SetName:        setName,
			},
			&expr.Reject{},
		},
		UserData: userData,
	})
}

func (this *Chain) AddAcceptInterfaceRule(interfaceName string, userData []byte) (*Rule, error) {
	if len(interfaceName) >= 16 {
		return nil, errors.New("invalid interface name '" + interfaceName + "'")
	}
	var ifname = make([]byte, 16)
	copy(ifname, interfaceName+"\x00")

	return this.AddRule(&RuleOptions{
		Exprs: []expr.Any{
			&expr.Meta{
				Key:      expr.MetaKeyIIFNAME,
				Register: 1,
			},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     ifname,
			},
			&expr.Verdict{
				Kind: expr.VerdictAccept,
			},
		},
		UserData: userData,
	})
}

func (this *Chain) GetRuleWithUserData(userData []byte) (*Rule, error) {
	rawRules, err := this.conn.Raw().GetRule(this.rawTable, this.rawChain)
	if err != nil {
		return nil, err
	}
	for _, rawRule := range rawRules {
		if bytes.Compare(rawRule.UserData, userData) == 0 {
			return NewRule(rawRule), nil
		}
	}
	return nil, ErrRuleNotFound
}

func (this *Chain) GetRules() ([]*Rule, error) {
	rawRules, err := this.conn.Raw().GetRule(this.rawTable, this.rawChain)
	if err != nil {
		return nil, err
	}
	var result = []*Rule{}
	for _, rawRule := range rawRules {
		result = append(result, NewRule(rawRule))
	}
	return result, nil
}

func (this *Chain) DeleteRule(rule *Rule) error {
	err := this.conn.Raw().DelRule(rule.Raw())
	if err != nil {
		return err
	}
	return this.conn.Commit()
}

func (this *Chain) Flush() error {
	this.conn.Raw().FlushChain(this.rawChain)
	return this.conn.Commit()
}

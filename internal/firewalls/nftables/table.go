// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.
//go:build linux

package nftables

import (
	"errors"
	nft "github.com/google/nftables"
	"github.com/iwind/TeaGo/types"
	"strings"
)

type Table struct {
	conn     *Conn
	rawTable *nft.Table
}

func NewTable(conn *Conn, rawTable *nft.Table) *Table {
	return &Table{
		conn:     conn,
		rawTable: rawTable,
	}
}

func (this *Table) Raw() *nft.Table {
	return this.rawTable
}

func (this *Table) Name() string {
	return this.rawTable.Name
}

func (this *Table) Family() TableFamily {
	return this.rawTable.Family
}

func (this *Table) GetChain(name string) (*Chain, error) {
	rawChains, err := this.conn.Raw().ListChains()
	if err != nil {
		return nil, err
	}
	for _, rawChain := range rawChains {
		// must compare table name
		if rawChain.Name == name && rawChain.Table.Name == this.rawTable.Name {
			return NewChain(this.conn, this.rawTable, rawChain), nil
		}
	}
	return nil, ErrChainNotFound
}

func (this *Table) AddChain(name string, chainPolicy *ChainPolicy) (*Chain, error) {
	if len(name) > MaxChainNameLength {
		return nil, errors.New("chain name too long (max " + types.String(MaxChainNameLength) + ")")
	}

	var rawChain = this.conn.Raw().AddChain(&nft.Chain{
		Name:     name,
		Table:    this.rawTable,
		Hooknum:  nft.ChainHookInput,
		Priority: nft.ChainPriorityFilter,
		Type:     nft.ChainTypeFilter,
		Policy:   chainPolicy,
	})

	err := this.conn.Commit()
	if err != nil {
		return nil, err
	}
	return NewChain(this.conn, this.rawTable, rawChain), nil
}

func (this *Table) AddAcceptChain(name string) (*Chain, error) {
	var policy = ChainPolicyAccept
	return this.AddChain(name, &policy)
}

func (this *Table) AddDropChain(name string) (*Chain, error) {
	var policy = ChainPolicyDrop
	return this.AddChain(name, &policy)
}

func (this *Table) DeleteChain(name string) error {
	chain, err := this.GetChain(name)
	if err != nil {
		if err == ErrChainNotFound {
			return nil
		}
		return err
	}
	this.conn.Raw().DelChain(chain.Raw())
	return this.conn.Commit()
}

func (this *Table) GetSet(name string) (*Set, error) {
	rawSet, err := this.conn.Raw().GetSetByName(this.rawTable, name)
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			return nil, ErrSetNotFound
		}
		return nil, err
	}

	return NewSet(this.conn, rawSet), nil
}

func (this *Table) AddSet(name string, options *SetOptions) (*Set, error) {
	if len(name) > MaxSetNameLength {
		return nil, errors.New("set name too long (max " + types.String(MaxSetNameLength) + ")")
	}

	if options == nil {
		options = &SetOptions{}
	}
	var rawSet = &nft.Set{
		Table:      this.rawTable,
		ID:         options.Id,
		Name:       name,
		Anonymous:  options.Anonymous,
		Constant:   options.Constant,
		Interval:   options.Interval,
		IsMap:      options.IsMap,
		HasTimeout: options.HasTimeout,
		Timeout:    options.Timeout,
		KeyType:    options.KeyType,
		DataType:   options.DataType,
	}
	err := this.conn.Raw().AddSet(rawSet, nil)
	if err != nil {
		return nil, err
	}

	err = this.conn.Commit()
	if err != nil {
		return nil, err
	}

	return NewSet(this.conn, rawSet), nil
}

func (this *Table) DeleteSet(name string) error {
	set, err := this.GetSet(name)
	if err != nil {
		if err == ErrSetNotFound {
			return nil
		}
		return err
	}

	this.conn.Raw().DelSet(set.Raw())
	return this.conn.Commit()
}

func (this *Table) Flush() error {
	this.conn.Raw().FlushTable(this.rawTable)
	return this.conn.Commit()
}

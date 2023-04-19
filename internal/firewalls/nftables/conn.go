// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.
//go:build linux

package nftables

import (
	"errors"
	nft "github.com/google/nftables"
	"github.com/iwind/TeaGo/types"
)

const MaxTableNameLength = 27

type Conn struct {
	rawConn *nft.Conn
}

func NewConn() (*Conn, error) {
	conn, err := nft.New()
	if err != nil {
		return nil, err
	}
	return &Conn{
		rawConn: conn,
	}, nil
}

func (this *Conn) Raw() *nft.Conn {
	return this.rawConn
}

func (this *Conn) GetTable(name string, family TableFamily) (*Table, error) {
	rawTables, err := this.rawConn.ListTables()
	if err != nil {
		return nil, err
	}

	for _, rawTable := range rawTables {
		if rawTable.Name == name && rawTable.Family == family {
			return NewTable(this, rawTable), nil
		}
	}

	return nil, ErrTableNotFound
}

func (this *Conn) AddTable(name string, family TableFamily) (*Table, error) {
	if len(name) > MaxTableNameLength {
		return nil, errors.New("table name too long (max " + types.String(MaxTableNameLength) + ")")
	}

	var rawTable = this.rawConn.AddTable(&nft.Table{
		Family: family,
		Name:   name,
	})

	err := this.Commit()
	if err != nil {
		return nil, err
	}

	return NewTable(this, rawTable), nil
}

func (this *Conn) AddIPv4Table(name string) (*Table, error) {
	return this.AddTable(name, TableFamilyIPv4)
}

func (this *Conn) AddIPv6Table(name string) (*Table, error) {
	return this.AddTable(name, TableFamilyIPv6)
}

func (this *Conn) DeleteTable(name string, family TableFamily) error {
	table, err := this.GetTable(name, family)
	if err != nil {
		if err == ErrTableNotFound {
			return nil
		}
		return err
	}
	this.rawConn.DelTable(table.Raw())
	return this.Commit()
}

func (this *Conn) Commit() error {
	return this.rawConn.Flush()
}

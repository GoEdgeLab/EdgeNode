// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.
//go:build linux

package nftables

import (
	nft "github.com/google/nftables"
)

type SetBatch struct {
	conn   *Conn
	rawSet *nft.Set
}

func (this *SetBatch) AddElement(key []byte, options *ElementOptions) error {
	var rawElement = nft.SetElement{
		Key: key,
	}
	if options != nil {
		rawElement.Timeout = options.Timeout
	}
	return this.conn.Raw().SetAddElements(this.rawSet, []nft.SetElement{
		rawElement,
	})
}

func (this *SetBatch) DeleteElement(key []byte) error {
	return this.conn.Raw().SetDeleteElements(this.rawSet, []nft.SetElement{
		{
			Key: key,
		},
	})
}

func (this *SetBatch) Commit() error {
	return this.conn.Commit()
}

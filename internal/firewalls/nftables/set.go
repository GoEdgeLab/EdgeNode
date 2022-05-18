// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.
//go:build linux
// +build linux

package nftables

import (
	"errors"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	nft "github.com/google/nftables"
	"net"
	"strings"
	"time"
)

const MaxSetNameLength = 15

type SetOptions struct {
	Id         uint32
	HasTimeout bool
	Timeout    time.Duration
	KeyType    SetDataType
	DataType   SetDataType
	Constant   bool
	Interval   bool
	Anonymous  bool
	IsMap      bool
}

type ElementOptions struct {
	Timeout time.Duration
}

type Set struct {
	conn   *Conn
	rawSet *nft.Set
	batch  *SetBatch
}

func NewSet(conn *Conn, rawSet *nft.Set) *Set {
	return &Set{
		conn:   conn,
		rawSet: rawSet,
		batch: &SetBatch{
			conn:   conn,
			rawSet: rawSet,
		},
	}
}

func (this *Set) Raw() *nft.Set {
	return this.rawSet
}

func (this *Set) Name() string {
	return this.rawSet.Name
}

func (this *Set) AddElement(key []byte, options *ElementOptions) error {
	var rawElement = nft.SetElement{
		Key: key,
	}
	if options != nil {
		rawElement.Timeout = options.Timeout
	}
	err := this.conn.Raw().SetAddElements(this.rawSet, []nft.SetElement{
		rawElement,
	})
	if err != nil {
		return err
	}

	err = this.conn.Commit()
	if err != nil {
		// retry if exists
		if strings.Contains(err.Error(), "file exists") {
			deleteErr := this.conn.Raw().SetDeleteElements(this.rawSet, []nft.SetElement{
				{
					Key: key,
				},
			})
			if deleteErr == nil {
				err = this.conn.Raw().SetAddElements(this.rawSet, []nft.SetElement{
					rawElement,
				})
				if err == nil {
					err = this.conn.Commit()
				}
			}
		}
	}

	return err
}

func (this *Set) AddIPElement(ip string, options *ElementOptions) error {
	var ipObj = net.ParseIP(ip)
	if ipObj == nil {
		return errors.New("invalid ip '" + ip + "'")
	}

	if utils.IsIPv4(ip) {
		return this.AddElement(ipObj.To4(), options)
	} else {
		return this.AddElement(ipObj.To16(), options)
	}
}

func (this *Set) DeleteElement(key []byte) error {
	err := this.conn.Raw().SetDeleteElements(this.rawSet, []nft.SetElement{
		{
			Key: key,
		},
	})
	if err != nil {
		return err
	}
	err = this.conn.Commit()
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			err = nil
		}
	}
	return err
}

func (this *Set) DeleteIPElement(ip string) error {
	var ipObj = net.ParseIP(ip)
	if ipObj == nil {
		return errors.New("invalid ip '" + ip + "'")
	}

	if utils.IsIPv4(ip) {
		return this.DeleteElement(ipObj.To4())
	} else {
		return this.DeleteElement(ipObj.To16())
	}
}

func (this *Set) Batch() *SetBatch {
	return this.batch
}

func (this *Set) GetIPElements() ([]string, error) {
	elements, err := this.conn.Raw().GetSetElements(this.rawSet)
	if err != nil {
		return nil, err
	}

	var result = []string{}
	for _, element := range elements {
		result = append(result, net.IP(element.Key).String())
	}
	return result, nil
}

// not work current time
/**func (this *Set) Flush() error {
	this.conn.Raw().FlushSet(this.rawSet)
	return this.conn.Commit()
}**/

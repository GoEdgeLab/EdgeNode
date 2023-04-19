// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.
//go:build linux

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

	expiration *Expiration
}

func NewSet(conn *Conn, rawSet *nft.Set) *Set {
	var set = &Set{
		conn:       conn,
		rawSet:     rawSet,
		expiration: nil,
		batch: &SetBatch{
			conn:   conn,
			rawSet: rawSet,
		},
	}

	// retrieve set elements to improve "delete" speed
	set.initElements()

	return set
}

func (this *Set) Raw() *nft.Set {
	return this.rawSet
}

func (this *Set) Name() string {
	return this.rawSet.Name
}

func (this *Set) AddElement(key []byte, options *ElementOptions, overwrite bool) error {
	// check if already exists
	if this.expiration != nil && !overwrite && this.expiration.Contains(key) {
		return nil
	}

	var expiresTime = time.Time{}
	var rawElement = nft.SetElement{
		Key: key,
	}
	if options != nil {
		rawElement.Timeout = options.Timeout

		if options.Timeout > 0 {
			expiresTime = time.UnixMilli(time.Now().UnixMilli() + options.Timeout.Milliseconds())
		}
	}
	err := this.conn.Raw().SetAddElements(this.rawSet, []nft.SetElement{
		rawElement,
	})
	if err != nil {
		return err
	}

	err = this.conn.Commit()
	if err == nil {
		if this.expiration != nil {
			this.expiration.Add(key, expiresTime)
		}
	} else {
		var isFileExistsErr = strings.Contains(err.Error(), "file exists")
		if !overwrite && isFileExistsErr {
			// ignore file exists error
			return nil
		}

		// retry if exists
		if overwrite && isFileExistsErr {
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
					if err == nil {
						if this.expiration != nil {
							this.expiration.Add(key, expiresTime)
						}
					}
				}
			}
		}
	}

	return err
}

func (this *Set) AddIPElement(ip string, options *ElementOptions, overwrite bool) error {
	var ipObj = net.ParseIP(ip)
	if ipObj == nil {
		return errors.New("invalid ip '" + ip + "'")
	}

	if utils.IsIPv4(ip) {
		return this.AddElement(ipObj.To4(), options, overwrite)
	} else {
		return this.AddElement(ipObj.To16(), options, overwrite)
	}
}

func (this *Set) DeleteElement(key []byte) error {
	// if set element does not exist, we return immediately
	if this.expiration != nil && !this.expiration.Contains(key) {
		return nil
	}

	err := this.conn.Raw().SetDeleteElements(this.rawSet, []nft.SetElement{
		{
			Key: key,
		},
	})
	if err != nil {
		return err
	}
	err = this.conn.Commit()
	if err == nil {
		if this.expiration != nil {
			this.expiration.Remove(key)
		}
	} else {
		if strings.Contains(err.Error(), "no such file or directory") {
			err = nil

			if this.expiration != nil {
				this.expiration.Remove(key)
			}
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

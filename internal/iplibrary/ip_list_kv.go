// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package iplibrary

import (
	"encoding/binary"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/TeaOSLab/EdgeNode/internal/utils/kvstore"
	"testing"
	"time"
)

type KVIPList struct {
	ipTable       *kvstore.Table[*pb.IPItem]
	versionsTable *kvstore.Table[int64]

	encoder *IPItemEncoder[*pb.IPItem]

	cleanTicker *time.Ticker

	isClosed bool

	offsetItemKey string
}

func NewKVIPList() (*KVIPList, error) {
	var db = &KVIPList{
		cleanTicker: time.NewTicker(24 * time.Hour),
		encoder:     &IPItemEncoder[*pb.IPItem]{},
	}
	err := db.init()
	return db, err
}

func (this *KVIPList) init() error {
	store, storeErr := kvstore.DefaultStore()
	if storeErr != nil {
		return storeErr
	}
	db, dbErr := store.NewDB("ip_list")
	if dbErr != nil {
		return dbErr
	}

	{
		table, err := kvstore.NewTable[*pb.IPItem]("ip_items", this.encoder)
		if err != nil {
			return err
		}
		this.ipTable = table

		err = table.AddFields("expiresAt")
		if err != nil {
			return err
		}

		db.AddTable(table)
	}

	{
		table, err := kvstore.NewTable[int64]("versions", kvstore.NewIntValueEncoder[int64]())
		if err != nil {
			return err
		}
		this.versionsTable = table
		db.AddTable(table)
	}

	goman.New(func() {
		events.OnClose(func() {
			_ = this.Close()
			this.cleanTicker.Stop()
		})

		for range this.cleanTicker.C {
			deleteErr := this.DeleteExpiredItems()
			if deleteErr != nil {
				remotelogs.Error("IP_LIST_DB", "clean expired items failed: "+deleteErr.Error())
			}
		}
	})

	return nil
}

// Name 数据库名称代号
func (this *KVIPList) Name() string {
	return "kvstore"
}

// DeleteExpiredItems 删除过期的条目
func (this *KVIPList) DeleteExpiredItems() error {
	if this.isClosed {
		return nil
	}

	for {
		var found bool
		var currentTime = fasttime.Now().Unix()
		err := this.ipTable.
			Query().
			FieldAsc("expiresAt").
			ForUpdate().
			Limit(1000).
			FindAll(func(tx *kvstore.Tx[*pb.IPItem], item kvstore.Item[*pb.IPItem]) (goNext bool, err error) {
				if !item.Value.IsDeleted && item.Value.ExpiredAt == 0 { // never expires
					return kvstore.Skip()
				}
				if item.Value.ExpiredAt < currentTime-7*86400 /** keep for 7 days **/ {
					err = tx.Delete(item.Key)
					if err != nil {
						return false, err
					}
					found = true
					return true, nil
				}

				found = false
				return false, nil
			})
		if err != nil {
			return err
		}
		if !found {
			break
		}
	}

	return nil
}

func (this *KVIPList) AddItem(item *pb.IPItem) error {
	if this.isClosed {
		return nil
	}

	// 先删除
	var key = this.encoder.EncodeKey(item)
	err := this.ipTable.Delete(key)
	if err != nil {
		return err
	}

	// 如果是删除，则不再创建新记录
	if item.IsDeleted {
		return this.UpdateMaxVersion(item.Version)
	}

	err = this.ipTable.Set(key, item)
	if err != nil {
		return err
	}

	return this.UpdateMaxVersion(item.Version)
}

func (this *KVIPList) ReadItems(offset int64, size int64) (items []*pb.IPItem, goNextLoop bool, err error) {
	if this.isClosed {
		return
	}

	err = this.ipTable.
		Query().
		Offset(this.offsetItemKey).
		Limit(int(size)).
		FindAll(func(tx *kvstore.Tx[*pb.IPItem], item kvstore.Item[*pb.IPItem]) (goNext bool, err error) {
			this.offsetItemKey = item.Key
			goNextLoop = true

			if !item.Value.IsDeleted {
				items = append(items, item.Value)
			}
			return true, nil
		})
	return
}

// ReadMaxVersion 读取当前最大版本号
func (this *KVIPList) ReadMaxVersion() (int64, error) {
	if this.isClosed {
		return 0, errors.New("database has been closed")
	}

	version, err := this.versionsTable.Get("version")
	if err != nil {
		if kvstore.IsNotFound(err) {
			return 0, nil
		}
		return 0, err
	}
	return version, nil
}

// UpdateMaxVersion 修改版本号
func (this *KVIPList) UpdateMaxVersion(version int64) error {
	if this.isClosed {
		return nil
	}

	return this.versionsTable.Set("version", version)
}

func (this *KVIPList) TestInspect(t *testing.T) error {
	return this.ipTable.
		Query().
		FindAll(func(tx *kvstore.Tx[*pb.IPItem], item kvstore.Item[*pb.IPItem]) (goNext bool, err error) {
			if len(item.Key) != 8 {
				return false, errors.New("invalid key '" + item.Key + "'")
			}

			t.Log(binary.BigEndian.Uint64([]byte(item.Key)), "=>", item.Value)
			return true, nil
		})
}

// Flush to disk
func (this *KVIPList) Flush() error {
	return this.ipTable.DB().Store().Flush()
}

func (this *KVIPList) Close() error {
	this.isClosed = true
	return nil
}

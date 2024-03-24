// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package caches

import (
	"errors"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/TeaOSLab/EdgeNode/internal/utils/kvstore"
	"github.com/cockroachdb/pebble"
	"github.com/iwind/TeaGo/types"
	"regexp"
	"strings"
	"testing"
)

type KVListFileStore struct {
	path     string
	rawStore *kvstore.Store

	// tables
	itemsTable *kvstore.Table[*Item]

	isReady bool
}

func NewKVListFileStore(path string) *KVListFileStore {
	return &KVListFileStore{
		path: path,
	}
}

func (this *KVListFileStore) Open() error {
	var reg = regexp.MustCompile(`^(.+)/([\w-]+)(\.store)$`)
	var matches = reg.FindStringSubmatch(this.path)
	if len(matches) != 4 {
		return errors.New("invalid path '" + this.path + "'")
	}
	var dir = matches[1]
	var name = matches[2]

	rawStore, err := kvstore.OpenStoreDir(dir, name)
	if err != nil {
		return err
	}
	this.rawStore = rawStore

	db, err := rawStore.NewDB("cache")
	if err != nil {
		return err
	}

	{
		table, tableErr := kvstore.NewTable[*Item]("items", NewItemKVEncoder[*Item]())
		if tableErr != nil {
			return tableErr
		}

		err = table.AddFields("staleAt", "key", "wildKey", "createdAt")
		if err != nil {
			return err
		}

		db.AddTable(table)
		this.itemsTable = table
	}

	this.isReady = true

	return nil
}

func (this *KVListFileStore) Path() string {
	return this.path
}

func (this *KVListFileStore) AddItem(hash string, item *Item) error {
	if !this.isReady {
		return nil
	}

	var currentTime = fasttime.Now().Unix()
	if item.ExpiresAt <= currentTime {
		return errors.New("invalid expires time '" + types.String(item.ExpiresAt) + "'")
	}
	if item.CreatedAt <= 0 {
		item.CreatedAt = currentTime
	}
	if item.StaleAt <= 0 {
		item.StaleAt = item.ExpiresAt + DefaultStaleCacheSeconds
	}
	return this.itemsTable.Set(hash, item)
}

func (this *KVListFileStore) ExistItem(hash string) (bool, error) {
	if !this.isReady {
		return false, nil
	}

	item, err := this.itemsTable.Get(hash)
	if err != nil {
		if kvstore.IsKeyNotFound(err) {
			return false, nil
		}
		return false, err
	}
	if item == nil {
		return false, nil
	}

	return item.ExpiresAt >= fasttime.NewFastTime().Unix(), nil
}

func (this *KVListFileStore) ExistQuickItem(hash string) (bool, error) {
	if !this.isReady {
		return false, nil
	}

	return this.itemsTable.Exist(hash)
}

func (this *KVListFileStore) RemoveItem(hash string) error {
	if !this.isReady {
		return nil
	}

	return this.itemsTable.Delete(hash)
}

func (this *KVListFileStore) RemoveAllItems() error {
	if !this.isReady {
		return nil
	}

	return this.itemsTable.Truncate()
}

func (this *KVListFileStore) PurgeItems(count int, callback func(hash string) error) (int, error) {
	if !this.isReady {
		return 0, nil
	}

	var countFound int
	var currentTime = fasttime.Now().Unix()
	var hashList []string
	err := this.itemsTable.
		Query().
		FieldAsc("staleAt").
		Limit(count).
		FindAll(func(tx *kvstore.Tx[*Item], item kvstore.Item[*Item]) (goNext bool, err error) {
			if item.Value == nil {
				return true, nil
			}
			if item.Value.StaleAt < currentTime {
				countFound++
				hashList = append(hashList, item.Key)
				return true, nil
			}
			return false, nil
		})
	if err != nil {
		return 0, err
	}

	// delete items
	if len(hashList) > 0 {
		txErr := this.itemsTable.WriteTx(func(tx *kvstore.Tx[*Item]) error {
			for _, hash := range hashList {
				deleteErr := tx.Delete(hash)
				if deleteErr != nil {
					return deleteErr
				}
			}
			return nil
		})
		if txErr != nil {
			return 0, txErr
		}

		for _, hash := range hashList {
			callbackErr := callback(hash)
			if callbackErr != nil {
				return 0, callbackErr
			}
		}
	}

	return countFound, nil
}

func (this *KVListFileStore) PurgeLFUItems(count int, callback func(hash string) error) error {
	if !this.isReady {
		return nil
	}

	var hashList []string
	err := this.itemsTable.
		Query().
		FieldAsc("createdAt").
		Limit(count).
		FindAll(func(tx *kvstore.Tx[*Item], item kvstore.Item[*Item]) (goNext bool, err error) {
			if item.Value != nil {
				hashList = append(hashList, item.Key)
			}
			return true, nil
		})
	if err != nil {
		return err
	}

	// delete items
	if len(hashList) > 0 {
		txErr := this.itemsTable.WriteTx(func(tx *kvstore.Tx[*Item]) error {
			for _, hash := range hashList {
				deleteErr := tx.Delete(hash)
				if deleteErr != nil {
					return deleteErr
				}
			}
			return nil
		})
		if txErr != nil {
			return txErr
		}

		for _, hash := range hashList {
			callbackErr := callback(hash)
			if callbackErr != nil {
				return callbackErr
			}
		}
	}

	return nil
}

func (this *KVListFileStore) CleanItemsWithPrefix(prefix string) error {
	if !this.isReady {
		return nil
	}

	if len(prefix) == 0 {
		return nil
	}

	var currentTime = fasttime.Now().Unix()

	var fieldOffset []byte
	const size = 1000
	for {
		var count int
		err := this.itemsTable.
			Query().
			FieldPrefix("key", prefix).
			FieldOffset(fieldOffset).
			Limit(size).
			ForUpdate().
			FindAll(func(tx *kvstore.Tx[*Item], item kvstore.Item[*Item]) (goNext bool, err error) {
				if item.Value == nil {
					return true, nil
				}

				count++
				fieldOffset = item.FieldKey

				if item.Value.CreatedAt >= currentTime {
					return true, nil
				}
				if item.Value.ExpiresAt == 0 {
					return true, nil
				}

				item.Value.ExpiresAt = 0
				item.Value.StaleAt = 0

				setErr := tx.Set(item.Key, item.Value) // TODO improve performance
				if setErr != nil {
					return false, setErr
				}

				return true, nil
			})
		if err != nil {
			return err
		}

		if count < size {
			break
		}
	}

	return nil
}

func (this *KVListFileStore) CleanItemsWithWildcardPrefix(prefix string) error {
	if !this.isReady {
		return nil
	}

	if len(prefix) == 0 {
		return nil
	}

	var currentTime = fasttime.Now().Unix()

	var fieldOffset []byte
	const size = 1000
	for {
		var count int
		err := this.itemsTable.
			Query().
			FieldPrefix("wildKey", prefix).
			FieldOffset(fieldOffset).
			Limit(size).
			ForUpdate().
			FindAll(func(tx *kvstore.Tx[*Item], item kvstore.Item[*Item]) (goNext bool, err error) {
				if item.Value == nil {
					return true, nil
				}

				count++
				fieldOffset = item.FieldKey

				if item.Value.CreatedAt >= currentTime {
					return true, nil
				}
				if item.Value.ExpiresAt == 0 {
					return true, nil
				}

				item.Value.ExpiresAt = 0
				item.Value.StaleAt = 0

				setErr := tx.Set(item.Key, item.Value) // TODO improve performance
				if setErr != nil {
					return false, setErr
				}

				return true, nil
			})
		if err != nil {
			return err
		}

		if count < size {
			break
		}
	}

	return nil
}

func (this *KVListFileStore) CleanItemsWithWildcardKey(key string) error {
	if !this.isReady {
		return nil
	}

	if len(key) == 0 {
		return nil
	}

	var currentTime = fasttime.Now().Unix()

	for _, realKey := range []string{key, key + SuffixAll} {
		var fieldOffset = append(this.itemsTable.FieldKey("wildKey"), '$')
		fieldOffset = append(fieldOffset, realKey...)
		const size = 1000

		var wildKey string
		if !strings.HasSuffix(realKey, SuffixAll) {
			wildKey = string(append([]byte(realKey), 0, 0))
		} else {
			wildKey = realKey
		}

		for {
			var count int
			err := this.itemsTable.
				Query().
				FieldPrefix("wildKey", wildKey).
				FieldOffset(fieldOffset).
				Limit(size).
				ForUpdate().
				FindAll(func(tx *kvstore.Tx[*Item], item kvstore.Item[*Item]) (goNext bool, err error) {
					if item.Value == nil {
						return true, nil
					}

					count++
					fieldOffset = item.FieldKey

					if item.Value.CreatedAt >= currentTime {
						return true, nil
					}
					if item.Value.ExpiresAt == 0 {
						return true, nil
					}

					item.Value.ExpiresAt = 0
					item.Value.StaleAt = 0

					setErr := tx.Set(item.Key, item.Value) // TODO improve performance
					if setErr != nil {
						return false, setErr
					}

					return true, nil
				})
			if err != nil {
				return err
			}

			if count < size {
				break
			}
		}
	}

	return nil
}

func (this *KVListFileStore) CountItems() (int64, error) {
	if !this.isReady {
		return 0, nil
	}

	return this.itemsTable.Count()
}

func (this *KVListFileStore) StatItems() (*Stat, error) {
	if !this.isReady {
		return &Stat{}, nil
	}

	var stat = &Stat{}

	err := this.itemsTable.
		Query().
		FindAll(func(tx *kvstore.Tx[*Item], item kvstore.Item[*Item]) (goNext bool, err error) {
			if item.Value != nil {
				stat.Size += item.Value.Size()
				stat.ValueSize += item.Value.BodySize
				stat.Count++
			}
			return true, nil
		})
	return stat, err
}

func (this *KVListFileStore) TestInspect(t *testing.T) error {
	if !this.isReady {
		return nil
	}

	it, err := this.rawStore.RawDB().NewIter(&pebble.IterOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = it.Close()
	}()

	for it.First(); it.Valid(); it.Next() {
		valueBytes, valueErr := it.ValueAndErr()
		if valueErr != nil {
			return valueErr
		}
		t.Log(string(it.Key()), "=>", string(valueBytes))
	}
	return nil
}

func (this *KVListFileStore) Close() error {
	this.isReady = false

	if this.rawStore != nil {
		return this.rawStore.Close()
	}

	return nil
}

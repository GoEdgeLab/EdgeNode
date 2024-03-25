// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package kvstore

import (
	"errors"
	"fmt"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/cockroachdb/pebble"
	"github.com/iwind/TeaGo/Tea"
	"io"
	"os"
	"strings"
	"sync"
)

const StoreSuffix = ".store"

type Store struct {
	name string

	path  string
	rawDB *pebble.DB

	isClosed bool

	dbs []*DB

	mu sync.Mutex
}

// NewStore create store with name
func NewStore(storeName string) (*Store, error) {
	if !IsValidName(storeName) {
		return nil, errors.New("invalid store name '" + storeName + "'")
	}

	var root = Tea.Root + "/data/stores"
	_, err := os.Stat(root)
	if err != nil && os.IsNotExist(err) {
		_ = os.MkdirAll(root, 0777)
	}

	return &Store{
		name: storeName,
		path: Tea.Root + "/data/stores/" + storeName + StoreSuffix,
	}, nil
}

func OpenStore(storeName string) (*Store, error) {
	store, err := NewStore(storeName)
	if err != nil {
		return nil, err
	}
	err = store.Open()
	if err != nil {
		return nil, err
	}

	return store, nil
}

func OpenStoreDir(dir string, storeName string) (*Store, error) {
	if !IsValidName(storeName) {
		return nil, errors.New("invalid store name '" + storeName + "'")
	}

	_, err := os.Stat(dir)
	if err != nil && os.IsNotExist(err) {
		_ = os.MkdirAll(dir, 0777)
	}

	dir = strings.TrimSuffix(dir, "/")

	var store = &Store{
		name: storeName,
		path: dir + "/" + storeName + StoreSuffix,
	}

	err = store.Open()
	if err != nil {
		return nil, err
	}
	return store, nil
}

func (this *Store) Open() error {
	var opt = &pebble.Options{
		Logger: NewLogger(),
	}

	var memoryMB = utils.SystemMemoryGB() * 1
	if memoryMB > 256 {
		memoryMB = 256
	}
	if memoryMB > 4 {
		opt.MemTableSize = uint64(memoryMB) << 20
	}

	rawDB, err := pebble.Open(this.path, opt)
	if err != nil {
		return err
	}
	this.rawDB = rawDB

	// events
	events.OnKey(events.EventQuit, fmt.Sprintf("kvstore_%p", this), func() {
		_ = this.Close()
	})
	events.OnKey(events.EventTerminated, fmt.Sprintf("kvstore_%p", this), func() {
		_ = this.Close()
	})

	return nil
}

func (this *Store) Set(keyBytes []byte, valueBytes []byte) error {
	return this.rawDB.Set(keyBytes, valueBytes, DefaultWriteOptions)
}

func (this *Store) Get(keyBytes []byte) (valueBytes []byte, closer io.Closer, err error) {
	return this.rawDB.Get(keyBytes)
}

func (this *Store) Delete(keyBytes []byte) error {
	return this.rawDB.Delete(keyBytes, DefaultWriteOptions)
}

func (this *Store) NewDB(dbName string) (*DB, error) {
	db, err := NewDB(this, dbName)
	if err != nil {
		return nil, err
	}

	this.mu.Lock()
	defer this.mu.Unlock()

	this.dbs = append(this.dbs, db)
	return db, nil
}

func (this *Store) RawDB() *pebble.DB {
	return this.rawDB
}

func (this *Store) Close() error {
	if this.isClosed {
		return nil
	}

	this.mu.Lock()
	var lastErr error
	for _, db := range this.dbs {
		err := db.Close()
		if err != nil {
			lastErr = err
		}
	}

	this.mu.Unlock()

	if this.rawDB != nil {
		this.isClosed = true
		err := this.rawDB.Close()
		if err != nil {
			return err
		}
	}

	return lastErr
}

func (this *Store) IsClosed() bool {
	return this.isClosed
}

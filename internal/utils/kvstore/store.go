// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package kvstore

import (
	"errors"
	"fmt"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	fsutils "github.com/TeaOSLab/EdgeNode/internal/utils/fs"
	memutils "github.com/TeaOSLab/EdgeNode/internal/utils/mem"
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

	path   string
	rawDB  *pebble.DB
	locker *fsutils.Locker

	isClosed bool

	dbs []*DB

	mu sync.Mutex
}

// NewStore create store with name
func NewStore(storeName string) (*Store, error) {
	if !IsValidName(storeName) {
		return nil, errors.New("invalid store name '" + storeName + "'")
	}

	var path = Tea.Root + "/data/stores/" + storeName + StoreSuffix
	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		_ = os.MkdirAll(path, 0777)
	}

	return &Store{
		name:   storeName,
		path:   path,
		locker: fsutils.NewLocker(path + "/.fs"),
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

var storeOnce = &sync.Once{}
var defaultSore *Store

func DefaultStore() (*Store, error) {
	if defaultSore != nil {
		return defaultSore, nil
	}

	var resultErr error
	storeOnce.Do(func() {
		store, err := NewStore("default")
		if err != nil {
			resultErr = fmt.Errorf("create default store failed: %w", err)
			remotelogs.Error("KV", resultErr.Error())
			return
		}
		err = store.Open()
		if err != nil {
			resultErr = fmt.Errorf("open default store failed: %w", err)
			remotelogs.Error("KV", resultErr.Error())
			return
		}
		defaultSore = store
	})

	return defaultSore, resultErr
}

func (this *Store) Open() error {
	err := this.locker.Lock()
	if err != nil {
		return err
	}

	var opt = &pebble.Options{
		Logger: NewLogger(),
	}

	var memoryMB = memutils.SystemMemoryGB() * 1
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
	events.OnClose(func() {
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

func (this *Store) Flush() error {
	return this.rawDB.Flush()
}

func (this *Store) Close() error {
	if this.isClosed {
		return nil
	}

	_ = this.locker.Release()

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

// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package kvstore

import (
	"errors"
	"sync"
)

type DB struct {
	store *Store

	name      string
	namespace string
	tableMap  map[string]TableInterface

	locker sync.RWMutex
}

func NewDB(store *Store, dbName string) (*DB, error) {
	if !IsValidName(dbName) {
		return nil, errors.New("invalid database name '" + dbName + "'")
	}

	return &DB{
		store:     store,
		name:      dbName,
		namespace: "$" + dbName + "$",
		tableMap:  map[string]TableInterface{},
	}, nil
}

func (this *DB) AddTable(table TableInterface) {
	table.SetNamespace([]byte(this.Namespace() + table.Name() + "$"))
	table.SetDB(this)

	this.locker.Lock()
	defer this.locker.Unlock()

	this.tableMap[table.Name()] = table
}

func (this *DB) Name() string {
	return this.name
}

func (this *DB) Namespace() string {
	return this.namespace
}

func (this *DB) Store() *Store {
	return this.store
}

func (this *DB) Close() error {
	this.locker.Lock()
	defer this.locker.Unlock()

	var lastErr error
	for _, table := range this.tableMap {
		err := table.Close()
		if err != nil {
			lastErr = err
		}
	}

	return lastErr
}

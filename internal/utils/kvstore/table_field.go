// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package kvstore

import (
	"errors"
)

func (this *Table[T]) AddField(fieldName string) error {
	if !IsValidName(fieldName) {
		return errors.New("invalid field name '" + fieldName + "'")
	}

	// check existence
	for _, field := range this.fieldNames {
		if field == fieldName {
			return nil
		}
	}

	this.fieldNames = append(this.fieldNames, fieldName)
	return nil
}

func (this *Table[T]) AddFields(fieldName ...string) error {
	for _, subFieldName := range fieldName {
		err := this.AddField(subFieldName)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *Table[T]) DropField(fieldName string) error {
	this.mu.Lock()
	defer this.mu.Unlock()

	var start = this.FieldKey(fieldName + "$")
	return this.db.store.rawDB.DeleteRange(start, append(start, 0xFF), DefaultWriteOptions)
}

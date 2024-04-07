// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package kvstore

import (
	"errors"
	"fmt"
	"github.com/cockroachdb/pebble"
)

var ErrTableNotFound = errors.New("table not found")
var ErrKeyTooLong = errors.New("too long key")
var ErrSkip = errors.New("skip") // skip count in iterator
var ErrTableClosed = errors.New("table closed")

func IsNotFound(err error) bool {
	return err != nil && errors.Is(err, pebble.ErrNotFound)
}

func IsSkipError(err error) bool {
	return err != nil && errors.Is(err, ErrSkip)
}

func Skip() (bool, error) {
	return true, ErrSkip
}

func NewTableClosedErr(tableName string) error {
	return fmt.Errorf("table '"+tableName+"' closed: %w", ErrTableClosed)
}

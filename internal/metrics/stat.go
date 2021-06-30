// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package metrics

import (
	"github.com/cespare/xxhash"
	"strconv"
)

type Stat struct {
	ServerId int64
	Keys     []string
	Hash     string
	Value    int64
	Time     string

	keysData []byte
}

func (this *Stat) Sum(version int, itemId int64) {
	this.Hash = strconv.FormatUint(xxhash.Sum64String(strconv.FormatInt(this.ServerId, 10)+"@"+string(this.keysData)+"@"+this.Time+"@"+strconv.Itoa(version)+"@"+strconv.FormatInt(itemId, 10)), 10)
}

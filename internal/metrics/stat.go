// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package metrics

import (
	"encoding/json"
	"github.com/cespare/xxhash"
	"strconv"
)

type Stat struct {
	ServerId int64
	Keys     []string
	Hash     string
	Value    int64
	Time     string
}

func SumStat(serverId int64, keys []string, time string, version int32, itemId int64) string {
	keysData, _ := json.Marshal(keys)
	return strconv.FormatUint(xxhash.Sum64String(strconv.FormatInt(serverId, 10)+"@"+string(keysData)+"@"+time+"@"+strconv.Itoa(int(version))+"@"+strconv.FormatInt(itemId, 10)), 10)
}

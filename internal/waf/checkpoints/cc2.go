// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/ttlcache"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
	"strings"
	"time"
)

var ccCache = ttlcache.NewCache()

// CC2Checkpoint 新的CC
type CC2Checkpoint struct {
	Checkpoint
}

func (this *CC2Checkpoint) RequestValue(req requests.Request, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	var keys = options.GetSlice("keys")
	var keyValues = []string{}
	for _, key := range keys {
		keyValues = append(keyValues, req.Format(types.String(key)))
	}
	if len(keyValues) == 0 {
		return
	}

	var period = options.GetInt64("period")
	if period <= 0 {
		period = 60
	}

	var threshold = options.GetInt64("threshold")
	if threshold <= 0 {
		threshold = 1000
	}

	value = ccCache.IncreaseInt64("WAF-CC-"+strings.Join(keyValues, "@"), 1, time.Now().Unix()+period, false)

	return
}

func (this *CC2Checkpoint) ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	return
}

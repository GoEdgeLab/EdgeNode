// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/TeaOSLab/EdgeNode/internal/waf/utils"
	"github.com/iwind/TeaGo/maps"
)

type RequestGeoCountryNameCheckpoint struct {
	Checkpoint
}

func (this *RequestGeoCountryNameCheckpoint) IsComposed() bool {
	return false
}

func (this *RequestGeoCountryNameCheckpoint) RequestValue(req requests.Request, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	value = req.Format("${geo.country.name}")
	return
}

func (this *RequestGeoCountryNameCheckpoint) ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	return this.RequestValue(req, param, options, ruleId)
}

func (this *RequestGeoCountryNameCheckpoint) CacheLife() utils.CacheLife {
	return utils.CacheLongLife
}

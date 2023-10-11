// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/TeaOSLab/EdgeNode/internal/waf/utils"
	"github.com/iwind/TeaGo/maps"
)

type RequestGeoProvinceNameCheckpoint struct {
	Checkpoint
}

func (this *RequestGeoProvinceNameCheckpoint) IsComposed() bool {
	return false
}

func (this *RequestGeoProvinceNameCheckpoint) RequestValue(req requests.Request, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	value = req.Format("${geo.province.name}")
	return
}

func (this *RequestGeoProvinceNameCheckpoint) ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	return this.RequestValue(req, param, options, ruleId)
}

func (this *RequestGeoProvinceNameCheckpoint) CacheLife() utils.CacheLife {
	return utils.CacheLongLife
}

// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/maps"
)

type RequestGeoCityNameCheckpoint struct {
	Checkpoint
}

func (this *RequestGeoCityNameCheckpoint) IsComposed() bool {
	return false
}

func (this *RequestGeoCityNameCheckpoint) RequestValue(req requests.Request, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	value = req.Format("${geo.city.name}")
	return
}

func (this *RequestGeoCityNameCheckpoint) ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map) (value interface{}, sysErr error, userErr error) {
	return this.RequestValue(req, param, options)
}

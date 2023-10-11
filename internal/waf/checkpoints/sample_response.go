package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/TeaOSLab/EdgeNode/internal/waf/utils"
	"github.com/iwind/TeaGo/maps"
)

// SampleResponseCheckpoint just a sample checkpoint, copy and change it for your new checkpoint
type SampleResponseCheckpoint struct {
	Checkpoint
}

func (this *SampleResponseCheckpoint) IsRequest() bool {
	return false
}

func (this *SampleResponseCheckpoint) RequestValue(req requests.Request, param string, options maps.Map, ruleId int64) (value any, sysErr error, userErr error) {
	return
}

func (this *SampleResponseCheckpoint) ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	return
}

func (this *SampleResponseCheckpoint) CacheLife() utils.CacheLife {
	return utils.CacheMiddleLife
}

// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package checkpoints

import (
	"fmt"
	"github.com/TeaOSLab/EdgeNode/internal/utils/counters"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/TeaOSLab/EdgeNode/internal/waf/utils"
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
	"path/filepath"
	"strings"
)

var commonFileExtensionsMap = map[string]zero.Zero{
	".ico":   zero.New(),
	".jpg":   zero.New(),
	".jpeg":  zero.New(),
	".gif":   zero.New(),
	".png":   zero.New(),
	".webp":  zero.New(),
	".woff2": zero.New(),
	".js":    zero.New(),
	".css":   zero.New(),
	".ttf":   zero.New(),
	".otf":   zero.New(),
	".fnt":   zero.New(),
	".svg":   zero.New(),
}

// CC2Checkpoint 新的CC
type CC2Checkpoint struct {
	Checkpoint
}

func (this *CC2Checkpoint) RequestValue(req requests.Request, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	var keys = options.GetSlice("keys")
	var keyValues = []string{}
	var hasRemoteAddr = false
	for _, key := range keys {
		if key == "${remoteAddr}" || key == "${rawRemoteAddr}" {
			hasRemoteAddr = true
		}
		keyValues = append(keyValues, req.Format(types.String(key)))
	}
	if len(keyValues) == 0 {
		return
	}

	var period = options.GetInt("period")
	if period <= 0 {
		period = 60
	} else if period > 7*86400 {
		period = 7 * 86400
	}

	/**var threshold = options.GetInt64("threshold")
	if threshold <= 0 {
		threshold = 1000
	}**/

	var ignoreCommonFiles = options.GetBool("ignoreCommonFiles")
	if ignoreCommonFiles {
		var rawReq = req.WAFRaw()
		if len(rawReq.Referer()) > 0 {
			var ext = filepath.Ext(rawReq.URL.Path)
			if len(ext) > 0 {
				_, ok := commonFileExtensionsMap[strings.ToLower(ext)]
				if ok {
					return
				}
			}
		}
	}

	var ccKey = "WAF-CC-" + types.String(ruleId) + "-" + strings.Join(keyValues, "@")
	value = counters.SharedCounter.IncreaseKey(ccKey, period)

	// 基于指纹统计
	var enableFingerprint = true
	if options.Has("enableFingerprint") && !options.GetBool("enableFingerprint") {
		enableFingerprint = false
	}
	if hasRemoteAddr && enableFingerprint {
		var fingerprint = req.WAFFingerprint()
		if len(fingerprint) > 0 {
			var fpKeyValues = []string{}
			for _, key := range keys {
				if key == "${remoteAddr}" || key == "${rawRemoteAddr}" {
					fpKeyValues = append(fpKeyValues, fmt.Sprintf("%x", fingerprint))
					continue
				}
				fpKeyValues = append(fpKeyValues, req.Format(types.String(key)))
			}
			var fpCCKey = "WAF-CC-" + types.String(ruleId) + "-" + strings.Join(fpKeyValues, "@")
			var fpValue = counters.SharedCounter.IncreaseKey(fpCCKey, period)
			if fpValue > value.(uint64) {
				value = fpValue
			}
		}
	}

	return
}

func (this *CC2Checkpoint) ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options, ruleId)
	}

	return
}

func (this *CC2Checkpoint) CacheLife() utils.CacheLife {
	return utils.CacheDisabled
}

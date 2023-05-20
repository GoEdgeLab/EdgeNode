// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package stats

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fnv"
	"github.com/mssola/useragent"
	"sync"
)

var SharedUserAgentParser = NewUserAgentParser()

// UserAgentParser UserAgent解析器
type UserAgentParser struct {
	parser *useragent.UserAgent

	cacheMap1     map[uint64]UserAgentParserResult
	cacheMap2     map[uint64]UserAgentParserResult
	maxCacheItems int

	cacheCursor int
	locker      sync.RWMutex
}

func NewUserAgentParser() *UserAgentParser {
	var parser = &UserAgentParser{
		parser:      &useragent.UserAgent{},
		cacheMap1:   map[uint64]UserAgentParserResult{},
		cacheMap2:   map[uint64]UserAgentParserResult{},
		cacheCursor: 0,
	}

	parser.init()
	return parser
}

func (this *UserAgentParser) init() {
	var maxCacheItems = 10_000
	var systemMemory = utils.SystemMemoryGB()
	if systemMemory >= 16 {
		maxCacheItems = 40_000
	} else if systemMemory >= 8 {
		maxCacheItems = 30_000
	} else if systemMemory >= 4 {
		maxCacheItems = 20_000
	}
	this.maxCacheItems = maxCacheItems
}

func (this *UserAgentParser) Parse(userAgent string) (result UserAgentParserResult) {
	// 限制长度
	if len(userAgent) == 0 || len(userAgent) > 256 {
		return
	}

	var userAgentKey = fnv.HashString(userAgent)

	this.locker.RLock()
	cacheResult, ok := this.cacheMap1[userAgentKey]
	if ok {
		this.locker.RUnlock()
		return cacheResult
	}

	cacheResult, ok = this.cacheMap2[userAgentKey]
	if ok {
		this.locker.RUnlock()
		return cacheResult
	}
	this.locker.RUnlock()

	this.locker.Lock()
	defer this.locker.Unlock()

	this.parser.Parse(userAgent)
	result.OS = this.parser.OSInfo()
	result.BrowserName, result.BrowserVersion = this.parser.Browser()
	result.IsMobile = this.parser.Mobile()

	// 忽略特殊字符
	if len(result.BrowserName) > 0 {
		for _, r := range result.BrowserName {
			if r == '$' || r == '"' || r == '\'' || r == '<' || r == '>' || r == ')' {
				return
			}
		}
	}

	if this.cacheCursor == 0 {
		this.cacheMap1[userAgentKey] = result
		if len(this.cacheMap1) >= this.maxCacheItems {
			this.cacheCursor = 1
			this.cacheMap2 = map[uint64]UserAgentParserResult{}
		}
	} else {
		this.cacheMap2[userAgentKey] = result
		if len(this.cacheMap2) >= this.maxCacheItems {
			this.cacheCursor = 0
			this.cacheMap1 = map[uint64]UserAgentParserResult{}
		}
	}

	return
}

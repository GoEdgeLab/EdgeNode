// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package stats

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/mssola/user_agent"
)

// UserAgentParser UserAgent解析器
// 只支持单线程
type UserAgentParser struct {
	parser *user_agent.UserAgent

	cacheMap1     map[string]UserAgentParserResult
	cacheMap2     map[string]UserAgentParserResult
	maxCacheItems int

	cacheCursor int
}

func NewUserAgentParser() *UserAgentParser {
	var parser = &UserAgentParser{
		parser:      &user_agent.UserAgent{},
		cacheMap1:   map[string]UserAgentParserResult{},
		cacheMap2:   map[string]UserAgentParserResult{},
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

	cacheResult, ok := this.cacheMap1[userAgent]
	if ok {
		return cacheResult
	}

	cacheResult, ok = this.cacheMap2[userAgent]
	if ok {
		return cacheResult
	}

	this.parser.Parse(userAgent)
	result.os = this.parser.OSInfo()
	result.browserName, result.browserVersion = this.parser.Browser()

	if this.cacheCursor == 0 {
		this.cacheMap1[userAgent] = result
		if len(this.cacheMap1) >= this.maxCacheItems {
			this.cacheCursor = 1
			this.cacheMap2 = map[string]UserAgentParserResult{}
		}
	} else {
		this.cacheMap2[userAgent] = result
		if len(this.cacheMap2) >= this.maxCacheItems {
			this.cacheCursor = 0
			this.cacheMap1 = map[string]UserAgentParserResult{}
		}
	}

	return
}

// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package stats

import (
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fnv"
	syncutils "github.com/TeaOSLab/EdgeNode/internal/utils/sync"
	"github.com/mssola/useragent"
	"sync"
	"time"
)

var SharedUserAgentParser = NewUserAgentParser()

const userAgentShardingCount = 8

// UserAgentParser UserAgent解析器
type UserAgentParser struct {
	cacheMaps [userAgentShardingCount]map[uint64]UserAgentParserResult
	pool      *sync.Pool
	mu        *syncutils.RWMutex

	maxCacheItems int
	gcTicker      *time.Ticker
	gcIndex       int
}

// NewUserAgentParser 获取新解析器
func NewUserAgentParser() *UserAgentParser {
	var parser = &UserAgentParser{
		pool: &sync.Pool{
			New: func() any {
				return &useragent.UserAgent{}
			},
		},
		cacheMaps: [userAgentShardingCount]map[uint64]UserAgentParserResult{},
		mu:        syncutils.NewRWMutex(userAgentShardingCount),
	}

	for i := 0; i < userAgentShardingCount; i++ {
		parser.cacheMaps[i] = map[uint64]UserAgentParserResult{}
	}

	parser.init()
	return parser
}

// 初始化
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

	this.gcTicker = time.NewTicker(5 * time.Second)
	goman.New(func() {
		for range this.gcTicker.C {
			this.GC()
		}
	})
}

// Parse 解析UserAgent
func (this *UserAgentParser) Parse(userAgent string) (result UserAgentParserResult) {
	// 限制长度
	if len(userAgent) == 0 || len(userAgent) > 256 {
		return
	}

	var userAgentKey = fnv.HashString(userAgent)
	var shardingIndex = int(userAgentKey % userAgentShardingCount)

	this.mu.RLock(shardingIndex)
	cacheResult, ok := this.cacheMaps[shardingIndex][userAgentKey]
	if ok {
		this.mu.RUnlock(shardingIndex)
		return cacheResult
	}
	this.mu.RUnlock(shardingIndex)

	var parser = this.pool.Get().(*useragent.UserAgent)
	parser.Parse(userAgent)
	result.OS = parser.OSInfo()
	result.BrowserName, result.BrowserVersion = parser.Browser()
	result.IsMobile = parser.Mobile()
	this.pool.Put(parser)

	// 忽略特殊字符
	if len(result.BrowserName) > 0 {
		for _, r := range result.BrowserName {
			if r == '$' || r == '"' || r == '\'' || r == '<' || r == '>' || r == ')' {
				return
			}
		}
	}

	this.mu.Lock(shardingIndex)
	this.cacheMaps[shardingIndex][userAgentKey] = result
	this.mu.Unlock(shardingIndex)

	return
}

// MaxCacheItems 读取能容纳的缓存最大数量
func (this *UserAgentParser) MaxCacheItems() int {
	return this.maxCacheItems
}

// Len 读取当前缓存数量
func (this *UserAgentParser) Len() int {
	var total = 0
	for i := 0; i < userAgentShardingCount; i++ {
		this.mu.RLock(i)
		total += len(this.cacheMaps[i])
		this.mu.RUnlock(i)
	}
	return total
}

// GC 回收多余的缓存
func (this *UserAgentParser) GC() {
	var total = this.Len()
	if total > this.maxCacheItems {
		for {
			var shardingIndex = this.gcIndex

			this.mu.Lock(shardingIndex)
			total -= len(this.cacheMaps[shardingIndex])
			this.cacheMaps[shardingIndex] = map[uint64]UserAgentParserResult{}
			this.gcIndex = (this.gcIndex + 1) % userAgentShardingCount
			this.mu.Unlock(shardingIndex)

			if total <= this.maxCacheItems {
				break
			}
		}
	}
}

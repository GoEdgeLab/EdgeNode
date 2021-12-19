package utils

import (
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/iwind/TeaGo/Tea"
	"time"
)

var BytePool1k = NewBytePool(20480, 1024)
var BytePool4k = NewBytePool(20480, 4*1024)
var BytePool16k = NewBytePool(40960, 16*1024)
var BytePool32k = NewBytePool(20480, 32*1024)

// BytePool pool for get byte slice
type BytePool struct {
	c       chan []byte
	maxSize int
	length  int
	hasNew  bool
}

// NewBytePool 创建新对象
func NewBytePool(maxSize, length int) *BytePool {
	if maxSize <= 0 {
		maxSize = 1024
	}
	if length <= 0 {
		length = 128
	}
	var pool = &BytePool{
		c:       make(chan []byte, maxSize),
		maxSize: maxSize,
		length:  length,
	}

	pool.init()

	return pool
}

// 初始化
func (this *BytePool) init() {
	var ticker = time.NewTicker(2 * time.Minute)
	if Tea.IsTesting() {
		ticker = time.NewTicker(5 * time.Second)
	}
	goman.New(func() {
		for range ticker.C {
			if this.hasNew {
				this.hasNew = false
				continue
			}

			this.Purge()
		}
	})
}

// Get 获取一个新的byte slice
func (this *BytePool) Get() (b []byte) {
	select {
	case b = <-this.c:
	default:
		b = make([]byte, this.length)
		this.hasNew = true
	}
	return
}

// Put 放回一个使用过的byte slice
func (this *BytePool) Put(b []byte) {
	if cap(b) != this.length {
		return
	}
	select {
	case this.c <- b:
	default:
		// 已达最大容量，则抛弃
	}
}

// Length 单个字节slice长度
func (this *BytePool) Length() int {
	return this.length
}

// Size 当前的数量
func (this *BytePool) Size() int {
	return len(this.c)
}

// Purge 清理
func (this *BytePool) Purge() {
	// 1%
	var count = len(this.c) / 100
	if count == 0 {
		return
	}

Loop:
	for i := 0; i < count; i++ {
		select {
		case <-this.c:
		default:
			break Loop
		}
	}
}

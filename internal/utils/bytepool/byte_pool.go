package bytepool

import (
	"sync"
)

var Pool1k = NewPool(1 << 10)
var Pool4k = NewPool(4 << 10)
var Pool16k = NewPool(16 << 10)
var Pool32k = NewPool(32 << 10)

type Buf struct {
	Bytes []byte
}

// Pool for get byte slice
type Pool struct {
	length  int
	rawPool *sync.Pool
}

// NewPool 创建新对象
func NewPool(length int) *Pool {
	if length < 0 {
		length = 1024
	}
	return &Pool{
		length: length,
		rawPool: &sync.Pool{
			New: func() any {
				return &Buf{
					Bytes: make([]byte, length),
				}
			},
		},
	}
}

// Get 获取一个新的byte slice
func (this *Pool) Get() *Buf {
	return this.rawPool.Get().(*Buf)
}

// Put 放回一个使用过的byte slice
func (this *Pool) Put(ptr *Buf) {
	this.rawPool.Put(ptr)
}

// Length 单个字节slice长度
func (this *Pool) Length() int {
	return this.length
}

package utils

var BytePool1024 = NewBytePool(20480, 1024)

// pool for get byte slice
type BytePool struct {
	c      chan []byte
	length int

	lastSize int
}

// 创建新对象
func NewBytePool(maxSize, length int) *BytePool {
	if maxSize <= 0 {
		maxSize = 1024
	}
	if length <= 0 {
		length = 128
	}
	pool := &BytePool{
		c:      make(chan []byte, maxSize),
		length: length,
	}
	return pool
}

// 获取一个新的byte slice
func (this *BytePool) Get() (b []byte) {
	select {
	case b = <-this.c:
	default:
		b = make([]byte, this.length)
	}
	return
}

// 放回一个使用过的byte slice
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

// 当前的数量
func (this *BytePool) Size() int {
	return len(this.c)
}

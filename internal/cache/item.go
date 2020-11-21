package cache

type Item struct {
	value     interface{}
	expiredAt int64
}

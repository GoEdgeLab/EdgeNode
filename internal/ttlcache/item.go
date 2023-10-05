package ttlcache

type Item[T any] struct {
	Value     T
	expiredAt int64
}

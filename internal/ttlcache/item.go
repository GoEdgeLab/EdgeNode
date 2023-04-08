package ttlcache

type Item struct {
	Value     any
	expiredAt int64
}

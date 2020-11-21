package ttlcache

import "github.com/dchest/siphash"

func HashKey(key []byte) uint64 {
	return siphash.Hash(0, 0, key)
}

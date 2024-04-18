package ttlcache

import "github.com/cespare/xxhash/v2"

func HashKeyBytes(key []byte) uint64 {
	return xxhash.Sum64(key)
}

func HashKeyString(key string) uint64 {
	return xxhash.Sum64String(key)
}

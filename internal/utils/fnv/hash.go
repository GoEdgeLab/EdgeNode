// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package fnv

const (
	offset64 uint64 = 14695981039346656037
	prime64  uint64 = 1099511628211
)

// Hash
// 非unique Hash
func Hash(key string) uint64 {
	var hash = offset64
	for _, b := range key {
		hash ^= uint64(b)
		hash *= prime64
	}
	return hash
}

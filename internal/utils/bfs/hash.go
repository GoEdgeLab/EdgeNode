// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package bfs

import (
	"fmt"
	stringutil "github.com/iwind/TeaGo/utils/string"
)

var HashLen = 32

// CheckHash check hash string format
func CheckHash(hash string) bool {
	if len(hash) != HashLen {
		return false
	}

	for _, b := range hash {
		if !((b >= '0' && b <= '9') || (b >= 'a' && b <= 'f')) {
			return false
		}
	}

	return true
}

func CheckHashErr(hash string) error {
	if CheckHash(hash) {
		return nil
	}
	return fmt.Errorf("check hash '%s' failed: %w", hash, ErrInvalidHash)
}

func Hash(s string) string {
	return stringutil.Md5(s)
}

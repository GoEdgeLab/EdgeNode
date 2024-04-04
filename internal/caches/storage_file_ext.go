// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .
//go:build !plus

package caches

func (this *FileStorage) tryMMAPReader(isPartial bool, estimatedSize int64, path string) (Reader, error) {
	// stub
	return nil, nil
}

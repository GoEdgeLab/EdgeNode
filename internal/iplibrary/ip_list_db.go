// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package iplibrary

import "github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"

type IPListDB interface {
	Name() string
	DeleteExpiredItems() error
	ReadMaxVersion() (int64, error)
	UpdateMaxVersion(version int64) error
	ReadItems(offset int64, size int64) (items []*pb.IPItem, goNext bool, err error)
	AddItem(item *pb.IPItem) error
}

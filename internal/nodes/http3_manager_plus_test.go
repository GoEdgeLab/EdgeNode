// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .
//go:build plus

package nodes_test

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/nodes"
	"testing"
	"time"
)

func TestHTTP3Manager_Update(t *testing.T) {
	var manager = nodes.NewHTTP3Manager()
	err := manager.Update(map[int64]*nodeconfigs.HTTP3Policy{
		1: {
			IsOn: true,
			Port: 443,
		},
		2: {
			IsOn: true,
			Port: 444,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	/**{
		err = manager.Update(map[int64]*nodeconfigs.HTTP3Policy{
			1: {
				IsOn: false,
				Port: 443,
			},
			2: {
				IsOn: true,
				Port: 445,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
	}**/

	time.Sleep(1 * time.Minute)
}

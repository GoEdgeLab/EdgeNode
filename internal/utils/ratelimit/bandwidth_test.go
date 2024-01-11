// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package ratelimit_test

import (
	"context"
	"github.com/TeaOSLab/EdgeNode/internal/utils/ratelimit"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"testing"
)

func TestBandwidth(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var bandwidth = ratelimit.NewBandwidth(32 << 10)
	bandwidth.Ack(context.Background(), 123)
	bandwidth.Ack(context.Background(), 16 << 10)
	bandwidth.Ack(context.Background(), 32 << 10)
}

func TestBandwidth_0(t *testing.T) {
	var bandwidth = ratelimit.NewBandwidth(0)
	bandwidth.Ack(context.Background(), 123)
	bandwidth.Ack(context.Background(), 123456)
}

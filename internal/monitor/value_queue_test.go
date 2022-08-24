// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package monitor

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	_ "github.com/iwind/TeaGo/bootstrap"
	"github.com/iwind/TeaGo/logs"
	"google.golang.org/grpc/status"
	"testing"
)

func TestValueQueue_RPC(t *testing.T) {
	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		t.Fatal(err)
	}
	_, err = rpcClient.NodeValueRPC.CreateNodeValue(rpcClient.Context(), &pb.CreateNodeValueRequest{})
	if err != nil {
		statusErr, ok := status.FromError(err)
		if ok {
			logs.Println(statusErr.Code())
		}
		t.Fatal(err)
	}
	t.Log("ok")
}

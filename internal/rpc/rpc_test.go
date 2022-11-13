// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package rpc_test

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	_ "github.com/iwind/TeaGo/bootstrap"
	timeutil "github.com/iwind/TeaGo/utils/time"
	"sync"
	"testing"
	"time"
)

func TestRPCConcurrentCall(t *testing.T) {
	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		t.Fatal(err)
	}

	var before = time.Now()
	defer func() {
		t.Log("cost:", time.Since(before).Seconds()*1000, "ms")
	}()

	var concurrent = 3

	var wg = sync.WaitGroup{}
	wg.Add(concurrent)

	for i := 0; i < concurrent; i++ {
		go func() {
			defer wg.Done()

			_, err = rpcClient.NodeRPC.FindCurrentNodeConfig(rpcClient.Context(), &pb.FindCurrentNodeConfigRequest{})
			if err != nil {
				t.Log(err)
			}
		}()
	}

	wg.Wait()
}

func TestRPC_Retry(t *testing.T) {
	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		t.Fatal(err)
	}

	var ticker = time.NewTicker(1 * time.Second)
	for range ticker.C {
		go func() {
			_, err = rpcClient.NodeRPC.FindCurrentNodeConfig(rpcClient.Context(), &pb.FindCurrentNodeConfigRequest{})
			if err != nil {
				t.Log(timeutil.Format("H:i:s"), err)
			} else {
				t.Log(timeutil.Format("H:i:s"), "success")
			}
		}()
	}
}

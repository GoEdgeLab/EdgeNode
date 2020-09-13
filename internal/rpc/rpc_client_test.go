package rpc

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/configs"
	_ "github.com/iwind/TeaGo/bootstrap"
	"testing"
	"time"
)

func TestRPCClient_NodeRPC(t *testing.T) {
	before := time.Now()
	defer func() {
		t.Log(time.Since(before).Seconds()*1000, "ms")
	}()
	config, err := configs.LoadAPIConfig()
	if err != nil {
		t.Fatal(err)
	}
	rpc, err := NewRPCClient(config)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := rpc.NodeRPC().ComposeNodeConfig(rpc.Context(), &pb.ComposeNodeConfigRequest{})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(resp)
}

func TestSharedRPC_Stream(t *testing.T) {
	config, err := configs.LoadAPIConfig()
	if err != nil {
		t.Fatal(err)
	}
	rpc, err := NewRPCClient(config)
	if err != nil {
		t.Fatal(err)
	}
	client, err := rpc.NodeRPC().NodeStream(rpc.Context())
	if err != nil {
		t.Fatal(err)
	}
	for {
		resp, err := client.Recv()
		if err != nil {
			t.Fatal(err)
		}
		t.Log("recv:", resp)
	}
}

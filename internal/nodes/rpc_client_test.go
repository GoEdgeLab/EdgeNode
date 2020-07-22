package nodes

import (
	"github.com/TeaOSLab/EdgeNode/internal/configs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc/node"
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
	resp, err := rpc.NodeRPC().Config(rpc.Context(), &node.ConfigRequest{
		NodeId: "123456",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(resp)
}

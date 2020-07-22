package nodes

import (
	"context"
	"errors"
	"github.com/TeaOSLab/EdgeNode/internal/configs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc/node"
	"github.com/iwind/TeaGo/rands"
	"google.golang.org/grpc"
)

type RPCClient struct {
	nodeClients []node.ServiceClient
}

func NewRPCClient(apiConfig *configs.APIConfig) (*RPCClient, error) {
	nodeClients := []node.ServiceClient{}

	conns := []*grpc.ClientConn{}
	for _, endpoint := range apiConfig.RPC.Endpoints {
		conn, err := grpc.Dial(endpoint, grpc.WithInsecure())
		if err != nil {
			return nil, err
		}
		conns = append(conns, conn)
	}
	if len(conns) == 0 {
		return nil, errors.New("[RPC]no available endpoints")
	}

	// node clients
	for _, conn := range conns {
		nodeClients = append(nodeClients, node.NewServiceClient(conn))
	}

	return &RPCClient{
		nodeClients: nodeClients,
	}, nil
}

func (this *RPCClient) NodeRPC() node.ServiceClient {
	if len(this.nodeClients) > 0 {
		return this.nodeClients[rands.Int(0, len(this.nodeClients)-1)]
	}
	return nil
}

func (this *RPCClient) Context() context.Context {
	return context.Background()
}

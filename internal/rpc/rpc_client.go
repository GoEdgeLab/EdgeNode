package rpc

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/configs"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/encrypt"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/rands"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"net/url"
	"time"
)

type RPCClient struct {
	apiConfig *configs.APIConfig
	conns     []*grpc.ClientConn
}

func NewRPCClient(apiConfig *configs.APIConfig) (*RPCClient, error) {
	if apiConfig == nil {
		return nil, errors.New("api config should not be nil")
	}

	conns := []*grpc.ClientConn{}
	for _, endpoint := range apiConfig.RPC.Endpoints {
		u, err := url.Parse(endpoint)
		if err != nil {
			return nil, errors.New("parse endpoint failed: " + err.Error())
		}
		var conn *grpc.ClientConn
		if u.Scheme == "http" {
			conn, err = grpc.Dial(u.Host, grpc.WithInsecure())
		} else if u.Scheme == "https" {
			conn, err = grpc.Dial(u.Host, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
				InsecureSkipVerify: true,
			})))
		} else {
			return nil, errors.New("parse endpoint failed: invalid scheme '" + u.Scheme + "'")
		}
		if err != nil {
			return nil, err
		}
		conns = append(conns, conn)
	}
	if len(conns) == 0 {
		return nil, errors.New("[RPC]no available endpoints")
	}

	return &RPCClient{
		apiConfig: apiConfig,
		conns:     conns,
	}, nil
}

func (this *RPCClient) NodeRPC() pb.NodeServiceClient {
	return pb.NewNodeServiceClient(this.pickConn())
}

func (this *RPCClient) NodeLogRPC() pb.NodeLogServiceClient {
	return pb.NewNodeLogServiceClient(this.pickConn())
}

func (this *RPCClient) HTTPAccessLogRPC() pb.HTTPAccessLogServiceClient {
	return pb.NewHTTPAccessLogServiceClient(this.pickConn())
}

func (this *RPCClient) APINodeRPC() pb.APINodeServiceClient {
	return pb.NewAPINodeServiceClient(this.pickConn())
}

// 节点上下文信息
func (this *RPCClient) Context() context.Context {
	ctx := context.Background()
	m := maps.Map{
		"timestamp": time.Now().Unix(),
		"type":      "node",
		"userId":    0,
	}
	method, err := encrypt.NewMethodInstance(teaconst.EncryptMethod, this.apiConfig.Secret, this.apiConfig.NodeId)
	if err != nil {
		utils.PrintError(err)
		return context.Background()
	}
	data, err := method.Encrypt(m.AsJSON())
	if err != nil {
		utils.PrintError(err)
		return context.Background()
	}
	token := base64.StdEncoding.EncodeToString(data)
	ctx = metadata.AppendToOutgoingContext(ctx, "nodeId", this.apiConfig.NodeId, "token", token)
	return ctx
}

// 集群上下文
func (this *RPCClient) ClusterContext(clusterId string, clusterSecret string) context.Context {
	ctx := context.Background()
	m := maps.Map{
		"timestamp": time.Now().Unix(),
		"type":      "cluster",
		"userId":    0,
	}
	method, err := encrypt.NewMethodInstance(teaconst.EncryptMethod, clusterSecret, clusterId)
	if err != nil {
		utils.PrintError(err)
		return context.Background()
	}
	data, err := method.Encrypt(m.AsJSON())
	if err != nil {
		utils.PrintError(err)
		return context.Background()
	}
	token := base64.StdEncoding.EncodeToString(data)
	ctx = metadata.AppendToOutgoingContext(ctx, "nodeId", clusterId, "token", token)
	return ctx
}

// 随机选择一个连接
func (this *RPCClient) pickConn() *grpc.ClientConn {
	if len(this.conns) == 0 {
		return nil
	}
	return this.conns[rands.Int(0, len(this.conns)-1)]
}

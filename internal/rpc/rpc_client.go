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
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"net/url"
	"sync"
	"time"
)

type RPCClient struct {
	apiConfig *configs.APIConfig
	conns     []*grpc.ClientConn

	locker sync.Mutex
}

func NewRPCClient(apiConfig *configs.APIConfig) (*RPCClient, error) {
	if apiConfig == nil {
		return nil, errors.New("api config should not be nil")
	}

	client := &RPCClient{
		apiConfig: apiConfig,
	}

	err := client.init()
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (this *RPCClient) NodeRPC() pb.NodeServiceClient {
	return pb.NewNodeServiceClient(this.pickConn())
}

func (this *RPCClient) NodeLogRPC() pb.NodeLogServiceClient {
	return pb.NewNodeLogServiceClient(this.pickConn())
}

func (this *RPCClient) NodeTaskRPC() pb.NodeTaskServiceClient {
	return pb.NewNodeTaskServiceClient(this.pickConn())
}

func (this *RPCClient) NodeValueRPC() pb.NodeValueServiceClient {
	return pb.NewNodeValueServiceClient(this.pickConn())
}

func (this *RPCClient) HTTPAccessLogRPC() pb.HTTPAccessLogServiceClient {
	return pb.NewHTTPAccessLogServiceClient(this.pickConn())
}

func (this *RPCClient) APINodeRPC() pb.APINodeServiceClient {
	return pb.NewAPINodeServiceClient(this.pickConn())
}

func (this *RPCClient) IPLibraryRPC() pb.IPLibraryServiceClient {
	return pb.NewIPLibraryServiceClient(this.pickConn())
}

func (this *RPCClient) RegionCountryRPC() pb.RegionCountryServiceClient {
	return pb.NewRegionCountryServiceClient(this.pickConn())
}

func (this *RPCClient) RegionProvinceRPC() pb.RegionProvinceServiceClient {
	return pb.NewRegionProvinceServiceClient(this.pickConn())
}

func (this *RPCClient) IPListRPC() pb.IPListServiceClient {
	return pb.NewIPListServiceClient(this.pickConn())
}

func (this *RPCClient) IPItemRPC() pb.IPItemServiceClient {
	return pb.NewIPItemServiceClient(this.pickConn())
}

func (this *RPCClient) FileRPC() pb.FileServiceClient {
	return pb.NewFileServiceClient(this.pickConn())
}

func (this *RPCClient) FileChunkRPC() pb.FileChunkServiceClient {
	return pb.NewFileChunkServiceClient(this.pickConn())
}

func (this *RPCClient) ACMEAuthenticationRPC() pb.ACMEAuthenticationServiceClient {
	return pb.NewACMEAuthenticationServiceClient(this.pickConn())
}

func (this *RPCClient) ServerRPC() pb.ServerServiceClient {
	return pb.NewServerServiceClient(this.pickConn())
}

func (this *RPCClient) ServerDailyStatRPC() pb.ServerDailyStatServiceClient {
	return pb.NewServerDailyStatServiceClient(this.pickConn())
}

// Context 节点上下文信息
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

// ClusterContext 集群上下文
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

// Close 关闭连接
func (this *RPCClient) Close() {
	for _, conn := range this.conns {
		_ = conn.Close()
	}
}

// UpdateConfig 修改配置
func (this *RPCClient) UpdateConfig(config *configs.APIConfig) error {
	this.apiConfig = config
	return this.init()
}

// 初始化
func (this *RPCClient) init() error {
	// 重新连接
	conns := []*grpc.ClientConn{}
	for _, endpoint := range this.apiConfig.RPC.Endpoints {
		u, err := url.Parse(endpoint)
		if err != nil {
			return errors.New("parse endpoint failed: " + err.Error())
		}
		var conn *grpc.ClientConn
		if u.Scheme == "http" {
			conn, err = grpc.Dial(u.Host, grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(128*1024*1024)))
		} else if u.Scheme == "https" {
			conn, err = grpc.Dial(u.Host, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
				InsecureSkipVerify: true,
			})), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(128*1024*1024)))
		} else {
			return errors.New("parse endpoint failed: invalid scheme '" + u.Scheme + "'")
		}
		if err != nil {
			return err
		}
		conns = append(conns, conn)
	}
	if len(conns) == 0 {
		return errors.New("[RPC]no available endpoints")
	}

	// 这里不需要加锁，防止和pickConn()冲突
	this.conns = conns
	return nil
}

// 随机选择一个连接
func (this *RPCClient) pickConn() *grpc.ClientConn {
	this.locker.Lock()
	defer this.locker.Unlock()

	// 检查连接状态
	if len(this.conns) > 0 {
		availableConns := []*grpc.ClientConn{}
		for _, state := range []connectivity.State{connectivity.Ready, connectivity.Idle, connectivity.Connecting} {
			for _, conn := range this.conns {
				if conn.GetState() == state {
					availableConns = append(availableConns, conn)
				}
			}
			if len(availableConns) > 0 {
				break
			}
		}

		if len(availableConns) > 0 {
			return availableConns[rands.Int(0, len(availableConns)-1)]
		}

		// 关闭
		for _, conn := range this.conns {
			_ = conn.Close()
		}
	}

	// 重新初始化
	err := this.init()
	if err != nil {
		// 错误提示已经在构造对象时打印过，所以这里不再重复打印
		return nil
	}

	if len(this.conns) == 0 {
		return nil
	}

	return this.conns[rands.Int(0, len(this.conns)-1)]
}

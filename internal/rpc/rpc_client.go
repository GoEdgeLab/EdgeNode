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
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
	"net/url"
	"sync"
	"time"
)

type RPCClient struct {
	apiConfig *configs.APIConfig
	conns     []*grpc.ClientConn

	locker sync.RWMutex

	NodeRPC                pb.NodeServiceClient
	NodeLogRPC             pb.NodeLogServiceClient
	NodeTaskRPC            pb.NodeTaskServiceClient
	NodeValueRPC           pb.NodeValueServiceClient
	HTTPAccessLogRPC       pb.HTTPAccessLogServiceClient
	HTTPCacheTaskKeyRPC    pb.HTTPCacheTaskKeyServiceClient
	APINodeRPC             pb.APINodeServiceClient
	IPLibraryArtifactRPC   pb.IPLibraryArtifactServiceClient
	IPListRPC              pb.IPListServiceClient
	IPItemRPC              pb.IPItemServiceClient
	FileRPC                pb.FileServiceClient
	FileChunkRPC           pb.FileChunkServiceClient
	ACMEAuthenticationRPC  pb.ACMEAuthenticationServiceClient
	ServerRPC              pb.ServerServiceClient
	ServerDailyStatRPC     pb.ServerDailyStatServiceClient
	ServerBandwidthStatRPC pb.ServerBandwidthStatServiceClient
	MetricStatRPC          pb.MetricStatServiceClient
	FirewallRPC            pb.FirewallServiceClient
	SSLCertRPC             pb.SSLCertServiceClient
	ScriptRPC              pb.ScriptServiceClient
}

func NewRPCClient(apiConfig *configs.APIConfig) (*RPCClient, error) {
	if apiConfig == nil {
		return nil, errors.New("api config should not be nil")
	}

	var client = &RPCClient{
		apiConfig: apiConfig,
	}

	// 初始化RPC实例
	client.NodeRPC = pb.NewNodeServiceClient(client)
	client.NodeLogRPC = pb.NewNodeLogServiceClient(client)
	client.NodeTaskRPC = pb.NewNodeTaskServiceClient(client)
	client.NodeValueRPC = pb.NewNodeValueServiceClient(client)
	client.HTTPAccessLogRPC = pb.NewHTTPAccessLogServiceClient(client)
	client.HTTPCacheTaskKeyRPC = pb.NewHTTPCacheTaskKeyServiceClient(client)
	client.APINodeRPC = pb.NewAPINodeServiceClient(client)
	client.IPLibraryArtifactRPC = pb.NewIPLibraryArtifactServiceClient(client)
	client.IPListRPC = pb.NewIPListServiceClient(client)
	client.IPItemRPC = pb.NewIPItemServiceClient(client)
	client.FileRPC = pb.NewFileServiceClient(client)
	client.FileChunkRPC = pb.NewFileChunkServiceClient(client)
	client.ACMEAuthenticationRPC = pb.NewACMEAuthenticationServiceClient(client)
	client.ServerRPC = pb.NewServerServiceClient(client)
	client.ServerDailyStatRPC = pb.NewServerDailyStatServiceClient(client)
	client.ServerBandwidthStatRPC = pb.NewServerBandwidthStatServiceClient(client)
	client.MetricStatRPC = pb.NewMetricStatServiceClient(client)
	client.FirewallRPC = pb.NewFirewallServiceClient(client)
	client.SSLCertRPC = pb.NewSSLCertServiceClient(client)
	client.ScriptRPC = pb.NewScriptServiceClient(client)

	err := client.init()
	if err != nil {
		return nil, err
	}

	return client, nil
}

// Context 节点上下文信息
func (this *RPCClient) Context() context.Context {
	var ctx = context.Background()
	var m = maps.Map{
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
	var token = base64.StdEncoding.EncodeToString(data)
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
	this.locker.Lock()

	for _, conn := range this.conns {
		_ = conn.Close()
	}

	this.locker.Unlock()
}

// UpdateConfig 修改配置
func (this *RPCClient) UpdateConfig(config *configs.APIConfig) error {
	this.apiConfig = config

	this.locker.Lock()
	err := this.init()
	this.locker.Unlock()
	return err
}

// 初始化
func (this *RPCClient) init() error {
	// 重新连接
	var conns = []*grpc.ClientConn{}
	for _, endpoint := range this.apiConfig.RPC.Endpoints {
		u, err := url.Parse(endpoint)
		if err != nil {
			return errors.New("parse endpoint failed: " + err.Error())
		}
		var conn *grpc.ClientConn
		var callOptions = grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(128*1024*1024),
			grpc.MaxCallSendMsgSize(128*1024*1024),
			grpc.UseCompressor(gzip.Name),
		)
		if u.Scheme == "http" {
			conn, err = grpc.Dial(u.Host, grpc.WithTransportCredentials(insecure.NewCredentials()), callOptions)
		} else if u.Scheme == "https" {
			conn, err = grpc.Dial(u.Host, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
				InsecureSkipVerify: true,
			})), callOptions)
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
		var availableConns = []*grpc.ClientConn{}
		for _, stateArray := range [][2]connectivity.State{
			{connectivity.Ready, connectivity.Idle}, // 优先Ready和Idle
			{connectivity.Connecting, connectivity.Connecting},
		} {
			for _, conn := range this.conns {
				var state = conn.GetState()
				if state == stateArray[0] || state == stateArray[1] {
					availableConns = append(availableConns, conn)
				}
			}
			if len(availableConns) > 0 {
				return this.randConn(availableConns)
			}
		}

		if len(availableConns) > 0 {
			return this.randConn(availableConns)
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

	return this.randConn(this.conns)
}

func (this *RPCClient) Invoke(ctx context.Context, method string, args interface{}, reply interface{}, opts ...grpc.CallOption) error {
	var conn = this.pickConn()
	if conn == nil {
		return errors.New("can not get available grpc connection")
	}
	return conn.Invoke(ctx, method, args, reply, opts...)
}
func (this *RPCClient) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	var conn = this.pickConn()
	if conn == nil {
		return nil, errors.New("can not get available grpc connection")
	}
	return conn.NewStream(ctx, desc, method, opts...)
}

func (this *RPCClient) randConn(conns []*grpc.ClientConn) *grpc.ClientConn {
	var l = len(conns)
	if l == 0 {
		return nil
	}
	if l == 1 {
		return conns[0]
	}
	return conns[rands.Int(0, l-1)]
}

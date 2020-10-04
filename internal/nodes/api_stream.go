package nodes

import (
	"encoding/json"
	"github.com/TeaOSLab/EdgeCommon/pkg/messageconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/TeaOSLab/EdgeNode/internal/errors"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/iwind/TeaGo/logs"
	"strconv"
	"time"
)

type APIStream struct {
	stream pb.NodeService_NodeStreamClient
}

func NewAPIStream() *APIStream {
	return &APIStream{}
}

func (this *APIStream) Start() {
	for {
		err := this.loop()
		if err != nil {
			logs.Println("[API STREAM]" + err.Error())
			time.Sleep(10 * time.Second)
			continue
		}
		time.Sleep(1 * time.Second)
	}
}

func (this *APIStream) loop() error {
	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return errors.Wrap(err)
	}
	nodeStream, err := rpcClient.NodeRPC().NodeStream(rpcClient.Context())
	if err != nil {
		return errors.Wrap(err)
	}
	this.stream = nodeStream
	for {
		message, err := nodeStream.Recv()
		if err != nil {
			return errors.Wrap(err)
		}

		// 处理消息
		switch message.Code {
		case messageconfigs.MessageCodeConnectedAPINode: // 连接API节点成功
			err = this.handleConnectedAPINode(message)
		case messageconfigs.MessageCodeWriteCache: // 写入缓存
			err = this.handleWriteCache(message)
		case messageconfigs.MessageCodeReadCache: // 读取缓存
			err = this.handleReadCache(message)
		default:
			err = this.handleUnknownMessage(message)
		}
		if err != nil {
			logs.Println("[API STREAM]handle message failed: " + err.Error())
		}
	}
}

// 连接API节点成功
func (this *APIStream) handleConnectedAPINode(message *pb.NodeStreamMessage) error {
	// 更改连接的APINode信息
	if len(message.DataJSON) == 0 {
		return nil
	}
	msg := &messageconfigs.ConnectedAPINodeMessage{}
	err := json.Unmarshal(message.DataJSON, msg)
	if err != nil {
		return errors.Wrap(err)
	}

	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return errors.Wrap(err)
	}

	_, err = rpcClient.NodeRPC().UpdateNodeConnectedAPINodes(rpcClient.Context(), &pb.UpdateNodeConnectedAPINodesRequest{ApiNodeIds: []int64{msg.APINodeId}})
	if err != nil {
		return errors.Wrap(err)
	}
	logs.Println("[API STREAM]connected to api node '" + strconv.FormatInt(msg.APINodeId, 10) + "'")
	return nil
}

// 写入缓存
func (this *APIStream) handleWriteCache(message *pb.NodeStreamMessage) error {
	msg := &messageconfigs.WriteCacheMessage{}
	err := json.Unmarshal(message.DataJSON, msg)
	if err != nil {
		this.replyFail(message.RequestId, "decode message data failed: "+err.Error())
		return err
	}

	cachePolicy := &serverconfigs.HTTPCachePolicy{}
	err = json.Unmarshal(msg.CachePolicyJSON, cachePolicy)
	if err != nil {
		this.replyFail(message.RequestId, "decode cache policy config failed: "+err.Error())
		return err
	}

	storage := caches.SharedManager.FindStorageWithPolicy(cachePolicy.Id)
	if storage == nil {
		storage = caches.SharedManager.NewStorageWithPolicy(cachePolicy)
		if storage == nil {
			this.replyFail(message.RequestId, "invalid storage type '"+cachePolicy.Type+"'")
			return nil
		}
		defer func() {
			storage.Stop()
		}()
		err = storage.Init()
		if err != nil {
			this.replyFail(message.RequestId, "storage init failed: "+err.Error())
			return err
		}
	}

	writer, err := storage.Open(msg.Key, time.Now().Unix()+msg.LifeSeconds)
	if err != nil {
		this.replyFail(message.RequestId, "prepare writing failed: "+err.Error())
		return err
	}

	defer func() {
		// 不用担心重复
		_ = writer.Close()
	}()

	err = writer.Write(msg.Value)
	if err != nil {
		this.replyFail(message.RequestId, "write failed: "+err.Error())
		return err
	}
	_ = writer.Close()

	this.replyOk(message.RequestId, "write ok")

	return nil
}

// 读取缓存
func (this *APIStream) handleReadCache(message *pb.NodeStreamMessage) error {
	msg := &messageconfigs.ReadCacheMessage{}
	err := json.Unmarshal(message.DataJSON, msg)
	if err != nil {
		this.replyFail(message.RequestId, "decode message data failed: "+err.Error())
		return err
	}
	cachePolicy := &serverconfigs.HTTPCachePolicy{}
	err = json.Unmarshal(msg.CachePolicyJSON, cachePolicy)
	if err != nil {
		this.replyFail(message.RequestId, "decode cache policy config failed: "+err.Error())
		return err
	}

	storage := caches.SharedManager.FindStorageWithPolicy(cachePolicy.Id)
	if storage == nil {
		storage = caches.SharedManager.NewStorageWithPolicy(cachePolicy)
		if storage == nil {
			this.replyFail(message.RequestId, "invalid storage type '"+cachePolicy.Type+"'")
			return nil
		}
		defer func() {
			storage.Stop()
		}()
		err = storage.Init()
		if err != nil {
			this.replyFail(message.RequestId, "storage init failed: "+err.Error())
			return err
		}
	}

	buf := make([]byte, 1024)
	size := 0
	err = storage.Read(msg.Key, buf, func(data []byte, expiredAt int64) {
		size += len(data)
	})
	if err != nil {
		if err == caches.ErrNotFound {
			this.replyFail(message.RequestId, "key not found")
			return nil
		}
		this.replyFail(message.RequestId, "read key failed: "+err.Error())
		return err
	}

	this.replyOk(message.RequestId, "value "+strconv.Itoa(size)+" bytes")

	return nil
}

// 处理未知消息
func (this *APIStream) handleUnknownMessage(message *pb.NodeStreamMessage) error {
	this.replyFail(message.RequestId, "unknown message code '"+message.Code+"'")
	return nil
}

// 回复失败
func (this *APIStream) replyFail(requestId int64, message string) {
	_ = this.stream.Send(&pb.NodeStreamMessage{RequestId: requestId, IsOk: false, Message: message})
}

// 回复成功
func (this *APIStream) replyOk(requestId int64, message string) {
	_ = this.stream.Send(&pb.NodeStreamMessage{RequestId: requestId, IsOk: true, Message: message})
}

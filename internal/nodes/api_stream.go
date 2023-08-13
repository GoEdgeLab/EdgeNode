package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/messageconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/TeaOSLab/EdgeNode/internal/configs"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/errors"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/firewalls"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	executils "github.com/TeaOSLab/EdgeNode/internal/utils/exec"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/maps"
	"net/url"
	"regexp"
	"runtime"
	"strconv"
	"time"
)

type APIStream struct {
	stream pb.NodeService_NodeStreamClient

	isQuiting  bool
	cancelFunc context.CancelFunc
}

func NewAPIStream() *APIStream {
	return &APIStream{}
}

func (this *APIStream) Start() {
	events.OnKey(events.EventQuit, this, func() {
		this.isQuiting = true
		if this.cancelFunc != nil {
			this.cancelFunc()
		}
	})
	for {
		if this.isQuiting {
			return
		}
		err := this.loop()
		if err != nil {
			if rpc.IsConnError(err) {
				remotelogs.Debug("API_STREAM", err.Error())
			} else {
				remotelogs.Warn("API_STREAM", err.Error())
			}
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

	ctx, cancelFunc := context.WithCancel(rpcClient.Context())
	this.cancelFunc = cancelFunc

	defer func() {
		cancelFunc()
	}()

	nodeStream, err := rpcClient.NodeRPC.NodeStream(ctx)
	if err != nil {
		if this.isQuiting {
			return nil
		}
		return err
	}
	this.stream = nodeStream

	for {
		if this.isQuiting {
			remotelogs.Println("API_STREAM", "quit")
			break
		}

		message, err := nodeStream.Recv()
		if err != nil {
			if this.isQuiting {
				remotelogs.Println("API_STREAM", "quit")
				return nil
			}
			return err
		}

		// 处理消息
		switch message.Code {
		case messageconfigs.MessageCodeConnectedAPINode: // 连接API节点成功
			err = this.handleConnectedAPINode(message)
		case messageconfigs.MessageCodeWriteCache: // 写入缓存
			err = this.handleWriteCache(message)
		case messageconfigs.MessageCodeReadCache: // 读取缓存
			err = this.handleReadCache(message)
		case messageconfigs.MessageCodeStatCache: // 统计缓存
			err = this.handleStatCache(message)
		case messageconfigs.MessageCodeCleanCache: // 清理缓存
			err = this.handleCleanCache(message)
		case messageconfigs.MessageCodeNewNodeTask: // 有新的任务
			err = this.handleNewNodeTask(message)
		case messageconfigs.MessageCodeCheckSystemdService: // 检查Systemd服务
			err = this.handleCheckSystemdService(message)
		case messageconfigs.MessageCodeCheckLocalFirewall: // 检查本地防火墙
			err = this.handleCheckLocalFirewall(message)
		case messageconfigs.MessageCodeChangeAPINode: // 修改API节点地址
			err = this.handleChangeAPINode(message)
		default:
			err = this.handleUnknownMessage(message)
		}
		if err != nil {
			remotelogs.Error("API_STREAM", "handle message failed: "+err.Error())
		}
	}

	return nil
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

	_, err = rpc.SharedRPC()
	if err != nil {
		return errors.Wrap(err)
	}

	remotelogs.Println("API_STREAM", "connected to api node '"+strconv.FormatInt(msg.APINodeId, 10)+"'")

	// 重新读取配置
	if nodeConfigUpdatedAt == 0 {
		select {
		case nodeConfigChangedNotify <- true:
		default:

		}
	}

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

	storage, shouldStop, err := this.cacheStorage(message, msg.CachePolicyJSON)
	if err != nil {
		return err
	}
	if shouldStop {
		defer func() {
			storage.Stop()
		}()
	}

	expiredAt := time.Now().Unix() + msg.LifeSeconds
	writer, err := storage.OpenWriter(msg.Key, expiredAt, 200, -1, int64(len(msg.Value)), -1, false)
	if err != nil {
		this.replyFail(message.RequestId, "prepare writing failed: "+err.Error())
		return err
	}

	// 写入一个空的Header
	_, err = writer.WriteHeader([]byte(":"))
	if err != nil {
		this.replyFail(message.RequestId, "write failed: "+err.Error())
		return err
	}

	// 写入数据
	_, err = writer.Write(msg.Value)
	if err != nil {
		this.replyFail(message.RequestId, "write failed: "+err.Error())
		return err
	}

	err = writer.Close()
	if err != nil {
		this.replyFail(message.RequestId, "write failed: "+err.Error())
		return err
	}
	storage.AddToList(&caches.Item{
		Type:       writer.ItemType(),
		Key:        msg.Key,
		ExpiredAt:  expiredAt,
		HeaderSize: writer.HeaderSize(),
		BodySize:   writer.BodySize(),
	})

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

	storage, shouldStop, err := this.cacheStorage(message, msg.CachePolicyJSON)
	if err != nil {
		return err
	}
	if shouldStop {
		defer func() {
			storage.Stop()
		}()
	}

	reader, err := storage.OpenReader(msg.Key, false, false)
	if err != nil {
		if err == caches.ErrNotFound {
			this.replyFail(message.RequestId, "key not found")
			return nil
		}
		this.replyFail(message.RequestId, "read key failed: "+err.Error())
		return nil
	}
	defer func() {
		_ = reader.Close()
	}()

	this.replyOk(message.RequestId, "value "+strconv.FormatInt(reader.BodySize(), 10)+" bytes")

	return nil
}

// 统计缓存
func (this *APIStream) handleStatCache(message *pb.NodeStreamMessage) error {
	msg := &messageconfigs.ReadCacheMessage{}
	err := json.Unmarshal(message.DataJSON, msg)
	if err != nil {
		this.replyFail(message.RequestId, "decode message data failed: "+err.Error())
		return err
	}

	storage, shouldStop, err := this.cacheStorage(message, msg.CachePolicyJSON)
	if err != nil {
		return err
	}
	if shouldStop {
		defer func() {
			storage.Stop()
		}()
	}

	stat, err := storage.Stat()
	if err != nil {
		this.replyFail(message.RequestId, "stat failed: "+err.Error())
		return err
	}

	sizeFormat := ""
	if stat.Size < 1024 {
		sizeFormat = strconv.FormatInt(stat.Size, 10) + " Bytes"
	} else if stat.Size < 1024*1024 {
		sizeFormat = fmt.Sprintf("%.2f KB", float64(stat.Size)/1024)
	} else if stat.Size < 1024*1024*1024 {
		sizeFormat = fmt.Sprintf("%.2f MB", float64(stat.Size)/1024/1024)
	} else {
		sizeFormat = fmt.Sprintf("%.2f GB", float64(stat.Size)/1024/1024/1024)
	}
	this.replyOk(message.RequestId, "size:"+sizeFormat+", count:"+strconv.Itoa(stat.Count))

	return nil
}

// 清理缓存
func (this *APIStream) handleCleanCache(message *pb.NodeStreamMessage) error {
	msg := &messageconfigs.ReadCacheMessage{}
	err := json.Unmarshal(message.DataJSON, msg)
	if err != nil {
		this.replyFail(message.RequestId, "decode message data failed: "+err.Error())
		return err
	}

	storage, shouldStop, err := this.cacheStorage(message, msg.CachePolicyJSON)
	if err != nil {
		return err
	}
	if shouldStop {
		defer func() {
			storage.Stop()
		}()
	}

	err = storage.CleanAll()
	if err != nil {
		this.replyFail(message.RequestId, "clean cache failed: "+err.Error())
		return err
	}

	this.replyOk(message.RequestId, "ok")

	return nil
}

// 处理配置变化
func (this *APIStream) handleNewNodeTask(message *pb.NodeStreamMessage) error {
	select {
	case nodeTaskNotify <- true:
	default:

	}
	this.replyOk(message.RequestId, "ok")
	return nil
}

// 检查Systemd服务
func (this *APIStream) handleCheckSystemdService(message *pb.NodeStreamMessage) error {
	systemctl, err := executils.LookPath("systemctl")
	if err != nil {
		this.replyFail(message.RequestId, "'systemctl' not found")
		return nil
	}
	if len(systemctl) == 0 {
		this.replyFail(message.RequestId, "'systemctl' not found")
		return nil
	}

	var shortName = teaconst.SystemdServiceName
	var cmd = executils.NewTimeoutCmd(10*time.Second, systemctl, "is-enabled", shortName)
	cmd.WithStdout()
	err = cmd.Run()
	if err != nil {
		this.replyFail(message.RequestId, "'systemctl' command error: "+err.Error())
		return nil
	}
	if cmd.Stdout() == "enabled" {
		this.replyOk(message.RequestId, "ok")
	} else {
		this.replyFail(message.RequestId, "not installed")
	}
	return nil
}

// 检查本地防火墙
func (this *APIStream) handleCheckLocalFirewall(message *pb.NodeStreamMessage) error {
	var dataMessage = &messageconfigs.CheckLocalFirewallMessage{}
	err := json.Unmarshal(message.DataJSON, dataMessage)
	if err != nil {
		this.replyFail(message.RequestId, "decode message data failed: "+err.Error())
		return nil
	}

	// nft
	if dataMessage.Name == "nftables" {
		if runtime.GOOS != "linux" {
			this.replyFail(message.RequestId, "not Linux system")
			return nil
		}

		nft, err := executils.LookPath("nft")
		if err != nil {
			this.replyFail(message.RequestId, "'nft' not found: "+err.Error())
			return nil
		}

		var cmd = executils.NewTimeoutCmd(10*time.Second, nft, "--version")
		cmd.WithStdout()
		err = cmd.Run()
		if err != nil {
			this.replyFail(message.RequestId, "get version failed: "+err.Error())
			return nil
		}

		var outputString = cmd.Stdout()
		var versionMatches = regexp.MustCompile(`nftables v([\d.]+)`).FindStringSubmatch(outputString)
		if len(versionMatches) <= 1 {
			this.replyFail(message.RequestId, "can not get nft version")
			return nil
		}
		var version = versionMatches[1]

		var result = maps.Map{
			"version": version,
		}

		var protectionConfig = sharedNodeConfig.DDoSProtection
		err = firewalls.SharedDDoSProtectionManager.Apply(protectionConfig)
		if err != nil {
			this.replyFail(message.RequestId, dataMessage.Name+" was installed, but apply DDoS protection config failed: "+err.Error())
		} else {
			this.replyOk(message.RequestId, string(result.AsJSON()))
		}
	} else {
		this.replyFail(message.RequestId, "invalid firewall name '"+dataMessage.Name+"'")
	}

	return nil
}

// 修改API地址
func (this *APIStream) handleChangeAPINode(message *pb.NodeStreamMessage) error {
	config, err := configs.LoadAPIConfig()
	if err != nil {
		this.replyFail(message.RequestId, "read config error: "+err.Error())
		return nil
	}

	var messageData = &messageconfigs.ChangeAPINodeMessage{}
	err = json.Unmarshal(message.DataJSON, messageData)
	if err != nil {
		this.replyFail(message.RequestId, "unmarshal message failed: "+err.Error())
		return nil
	}

	_, err = url.Parse(messageData.Addr)
	if err != nil {
		this.replyFail(message.RequestId, "invalid new api node address: '"+messageData.Addr+"'")
		return nil
	}

	config.RPCEndpoints = []string{messageData.Addr}

	// 保存到文件
	err = config.WriteFile(Tea.ConfigFile(configs.ConfigFileName))
	if err != nil {
		this.replyFail(message.RequestId, "save config file failed: "+err.Error())
		return nil
	}

	this.replyOk(message.RequestId, "")

	goman.New(func() {
		// 延后生效，防止变更前的API无法读取到状态
		time.Sleep(1 * time.Second)

		rpcClient, err := rpc.SharedRPC()
		if err != nil {
			remotelogs.Error("API_STREAM", "change rpc endpoint to '"+
				messageData.Addr+"' failed: "+err.Error())
			return
		}

		rpcClient.Close()

		err = rpcClient.UpdateConfig(config)
		if err != nil {
			remotelogs.Error("API_STREAM", "change rpc endpoint to '"+
				messageData.Addr+"' failed: "+err.Error())
			return
		}

		remotelogs.Println("API_STREAM", "change rpc endpoint to '"+
			messageData.Addr+"' successfully")
	})

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

// 回复成功并包含数据
func (this *APIStream) replyOkData(requestId int64, message string, dataJSON []byte) {
	_ = this.stream.Send(&pb.NodeStreamMessage{RequestId: requestId, IsOk: true, Message: message, DataJSON: dataJSON})
}

// 获取缓存存取对象
func (this *APIStream) cacheStorage(message *pb.NodeStreamMessage, cachePolicyJSON []byte) (storage caches.StorageInterface, shouldStop bool, err error) {
	cachePolicy := &serverconfigs.HTTPCachePolicy{}
	err = json.Unmarshal(cachePolicyJSON, cachePolicy)
	if err != nil {
		this.replyFail(message.RequestId, "decode cache policy config failed: "+err.Error())
		return nil, false, err
	}

	storage = caches.SharedManager.FindStorageWithPolicy(cachePolicy.Id)
	if storage == nil {
		storage = caches.SharedManager.NewStorageWithPolicy(cachePolicy)
		if storage == nil {
			this.replyFail(message.RequestId, "invalid storage type '"+cachePolicy.Type+"'")
			return nil, false, errors.New("invalid storage type '" + cachePolicy.Type + "'")
		}
		err = storage.Init()
		if err != nil {
			this.replyFail(message.RequestId, "storage init failed: "+err.Error())
			return nil, false, err
		}
		shouldStop = true
	}

	return
}

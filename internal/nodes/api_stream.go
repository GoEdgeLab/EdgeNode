package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/messageconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/errors"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/logs"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type APIStream struct {
	stream pb.NodeService_NodeStreamClient
}

func NewAPIStream() *APIStream {
	return &APIStream{}
}

func (this *APIStream) Start() {
	isQuiting := false
	events.On(events.EventQuit, func() {
		isQuiting = true
	})
	for {
		if isQuiting {
			return
		}
		err := this.loop()
		if err != nil {
			remotelogs.Error("API_STREAM", err.Error())
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
	isQuiting := false
	ctx, cancelFunc := context.WithCancel(rpcClient.Context())
	nodeStream, err := rpcClient.NodeRPC().NodeStream(ctx)
	events.On(events.EventQuit, func() {
		isQuiting = true

		remotelogs.Println("API_STREAM", "quiting")
		if nodeStream != nil {
			cancelFunc()
		}
	})
	if err != nil {
		if isQuiting {
			return nil
		}
		return errors.Wrap(err)
	}
	this.stream = nodeStream

	for {
		if isQuiting {
			logs.Println("API_STREAM", "quit")
			break
		}

		message, err := nodeStream.Recv()
		if err != nil {
			if isQuiting {
				remotelogs.Println("API_STREAM", "quit")
				return nil
			}
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
		case messageconfigs.MessageCodeStatCache: // 统计缓存
			err = this.handleStatCache(message)
		case messageconfigs.MessageCodeCleanCache: // 清理缓存
			err = this.handleCleanCache(message)
		case messageconfigs.MessageCodePurgeCache: // 删除缓存
			err = this.handlePurgeCache(message)
		case messageconfigs.MessageCodePreheatCache: // 预热缓存
			err = this.handlePreheatCache(message)
		case messageconfigs.MessageCodeNewNodeTask: // 有新的任务
			err = this.handleNewNodeTask(message)
		case messageconfigs.MessageCodeCheckSystemdService: // 检查Systemd服务
			err = this.handleCheckSystemdService(message)
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

	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return errors.Wrap(err)
	}

	_, err = rpcClient.NodeRPC().UpdateNodeConnectedAPINodes(rpcClient.Context(), &pb.UpdateNodeConnectedAPINodesRequest{ApiNodeIds: []int64{msg.APINodeId}})
	if err != nil {
		return errors.Wrap(err)
	}
	remotelogs.Println("API_STREAM", "connected to api node '"+strconv.FormatInt(msg.APINodeId, 10)+"'")
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
	writer, err := storage.OpenWriter(msg.Key, expiredAt, 200)
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

	reader, err := storage.OpenReader(msg.Key)
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

// 删除缓存
func (this *APIStream) handlePurgeCache(message *pb.NodeStreamMessage) error {
	msg := &messageconfigs.PurgeCacheMessage{}
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

	err = storage.Purge(msg.Keys, msg.Type)
	if err != nil {
		this.replyFail(message.RequestId, "purge keys failed: "+err.Error())
		return err
	}

	this.replyOk(message.RequestId, "ok")

	return nil
}

// 预热缓存
func (this *APIStream) handlePreheatCache(message *pb.NodeStreamMessage) error {
	msg := &messageconfigs.PreheatCacheMessage{}
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

	if len(msg.Keys) == 0 {
		this.replyOk(message.RequestId, "ok")
		return nil
	}

	wg := sync.WaitGroup{}
	wg.Add(len(msg.Keys))
	client := http.Client{} // TODO 可以设置请求超时事件
	errorMessages := []string{}
	locker := sync.Mutex{}
	for _, key := range msg.Keys {
		go func(key string) {
			defer wg.Done()

			req, err := http.NewRequest(http.MethodGet, key, nil)
			if err != nil {
				locker.Lock()
				errorMessages = append(errorMessages, "invalid url: "+key+": "+err.Error())
				locker.Unlock()
				return
			}
			// TODO 可以在管理界面自定义Header
			req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/85.0.4183.121 Safari/537.36")
			req.Header.Set("Accept-Encoding", "gzip, deflate, br") // TODO 这里需要记录下缓存是否为gzip的
			resp, err := client.Do(req)
			if err != nil {
				locker.Lock()
				errorMessages = append(errorMessages, "request failed: "+key+": "+err.Error())
				locker.Unlock()
				return
			}

			if resp.StatusCode != 200 {
				locker.Lock()
				errorMessages = append(errorMessages, "request failed: "+key+": status code '"+strconv.Itoa(resp.StatusCode)+"'")
				locker.Unlock()
				return
			}

			defer func() {
				_ = resp.Body.Close()
			}()

			// 检查最大内容长度
			// TODO 需要解决Chunked Transfer Encoding的长度判断问题
			maxSize := storage.Policy().MaxSizeBytes()
			if maxSize > 0 && resp.ContentLength > maxSize {
				locker.Lock()
				errorMessages = append(errorMessages, "request failed: the content is too larger than policy setting")
				locker.Unlock()
				return
			}

			expiredAt := time.Now().Unix() + 8600
			writer, err := storage.OpenWriter(key, expiredAt, 200) // TODO 可以设置缓存过期时间
			if err != nil {
				locker.Lock()
				errorMessages = append(errorMessages, "open cache writer failed: "+key+": "+err.Error())
				locker.Unlock()
				return
			}

			buf := make([]byte, 16*1024)
			isClosed := false

			// 写入Header
			for k, v := range resp.Header {
				for _, v1 := range v {
					_, err = writer.WriteHeader([]byte(k + ":" + v1 + "\n"))
					if err != nil {
						locker.Lock()
						errorMessages = append(errorMessages, "write failed: "+key+": "+err.Error())
						locker.Unlock()
						return
					}
				}
			}

			// 写入Body
			for {
				n, err := resp.Body.Read(buf)
				if n > 0 {
					_, writerErr := writer.Write(buf[:n])
					if writerErr != nil {
						locker.Lock()
						errorMessages = append(errorMessages, "write failed: "+key+": "+writerErr.Error())
						locker.Unlock()
						break
					}
				}
				if err != nil {
					if err == io.EOF {

						err = writer.Close()
						if err == nil {
							storage.AddToList(&caches.Item{
								Type:      writer.ItemType(),
								Key:       key,
								ExpiredAt: expiredAt,
							})
						}
						isClosed = true
					} else {
						locker.Lock()
						errorMessages = append(errorMessages, "read url failed: "+key+": "+err.Error())
						locker.Unlock()
					}
					break
				}
			}

			if !isClosed {
				_ = writer.Close()
			}
		}(key)
	}
	wg.Wait()

	if len(errorMessages) > 0 {
		this.replyFail(message.RequestId, strings.Join(errorMessages, ", "))
		return nil
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
	systemctl, err := exec.LookPath("systemctl")
	if err != nil {
		this.replyFail(message.RequestId, "'systemctl' not found")
		return nil
	}
	if len(systemctl) == 0 {
		this.replyFail(message.RequestId, "'systemctl' not found")
		return nil
	}

	cmd := utils.NewCommandExecutor()
	shortName := teaconst.SystemdServiceName
	cmd.Add(systemctl, "is-enabled", shortName)
	output, err := cmd.Run()
	if err != nil {
		this.replyFail(message.RequestId, "'systemctl' command error: "+err.Error())
		return nil
	}
	if output == "enabled" {
		this.replyOk(message.RequestId, "ok")
	} else {
		this.replyFail(message.RequestId, "not installed")
	}
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

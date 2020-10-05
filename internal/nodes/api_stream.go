package nodes

import (
	"encoding/json"
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/messageconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/TeaOSLab/EdgeNode/internal/errors"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/iwind/TeaGo/logs"
	"io"
	"net/http"
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
		case messageconfigs.MessageCodeStatCache: // 统计缓存
			err = this.handleStatCache(message)
		case messageconfigs.MessageCodeCleanCache: // 清理缓存
			err = this.handleCleanCache(message)
		case messageconfigs.MessageCodePurgeCache: // 删除缓存
			err = this.handlePurgeCache(message)
		case messageconfigs.MessageCodePreheatCache: // 预热缓存
			err = this.handlePreheatCache(message)
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
	writer, err := storage.Open(msg.Key, expiredAt)
	if err != nil {
		this.replyFail(message.RequestId, "prepare writing failed: "+err.Error())
		return err
	}

	_, err = writer.Write(msg.Value)
	if err != nil {
		_ = writer.Discard()
		this.replyFail(message.RequestId, "write failed: "+err.Error())
		return err
	}
	err = writer.Close()
	if err == nil {
		storage.AddToList(&caches.Item{
			Key:       msg.Key,
			ExpiredAt: expiredAt,
		})
	}

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

	buf := make([]byte, 1024)
	size := 0
	err = storage.Read(msg.Key, buf, func(data []byte, valueSize int64, expiredAt int64, isEOF bool) {
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

	err = storage.Purge(msg.Keys)
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

			defer func() {
				_ = resp.Body.Close()
			}()

			expiredAt := time.Now().Unix() + 8600
			writer, err := storage.Open(key, expiredAt) // TODO 可以设置缓存过期事件
			if err != nil {
				locker.Lock()
				errorMessages = append(errorMessages, "open cache writer failed: "+key+": "+err.Error())
				locker.Unlock()
				return
			}

			buf := make([]byte, 16*1024)
			isClosed := false
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
			return nil, false, err
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

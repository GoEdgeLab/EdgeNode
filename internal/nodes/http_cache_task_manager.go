// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package nodes

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/TeaOSLab/EdgeNode/internal/compressions"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	connutils "github.com/TeaOSLab/EdgeNode/internal/utils/conns"
	"github.com/iwind/TeaGo/Tea"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

func init() {
	if !teaconst.IsMain {
		return
	}

	events.On(events.EventStart, func() {
		goman.New(func() {
			SharedHTTPCacheTaskManager.Start()
		})
	})
}

var SharedHTTPCacheTaskManager = NewHTTPCacheTaskManager()

// HTTPCacheTaskManager 缓存任务管理
type HTTPCacheTaskManager struct {
	ticker      *time.Ticker
	protocolReg *regexp.Regexp

	timeoutClientMap map[time.Duration]*http.Client // timeout seconds=> *http.Client
	locker           sync.Mutex

	taskQueue chan *pb.PurgeServerCacheRequest
}

func NewHTTPCacheTaskManager() *HTTPCacheTaskManager {
	var duration = 30 * time.Second
	if Tea.IsTesting() {
		duration = 10 * time.Second
	}

	return &HTTPCacheTaskManager{
		ticker:           time.NewTicker(duration),
		protocolReg:      regexp.MustCompile(`^(?i)(http|https)://`),
		taskQueue:        make(chan *pb.PurgeServerCacheRequest, 1024),
		timeoutClientMap: make(map[time.Duration]*http.Client),
	}
}

func (this *HTTPCacheTaskManager) Start() {
	// task queue
	goman.New(func() {
		rpcClient, _ := rpc.SharedRPC()

		if rpcClient != nil {
			for taskReq := range this.taskQueue {
				_, err := rpcClient.ServerRPC.PurgeServerCache(rpcClient.Context(), taskReq)
				if err != nil {
					remotelogs.Error("HTTP_CACHE_TASK_MANAGER", "create purge task failed: "+err.Error())
				}
			}
		}
	})

	// Loop
	for range this.ticker.C {
		err := this.Loop()
		if err != nil {
			remotelogs.Error("HTTP_CACHE_TASK_MANAGER", "execute task failed: "+err.Error())
		}
	}
}

func (this *HTTPCacheTaskManager) Loop() error {
	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return err
	}

	resp, err := rpcClient.HTTPCacheTaskKeyRPC.FindDoingHTTPCacheTaskKeys(rpcClient.Context(), &pb.FindDoingHTTPCacheTaskKeysRequest{})
	if err != nil {
		// 忽略连接错误
		if rpc.IsConnError(err) {
			return nil
		}
		return err
	}

	var keys = resp.HttpCacheTaskKeys
	if len(keys) == 0 {
		return nil
	}

	var pbResults = []*pb.UpdateHTTPCacheTaskKeysStatusRequest_KeyResult{}

	var taskGroup = goman.NewTaskGroup()
	for _, key := range keys {
		var taskKey = key
		taskGroup.Run(func() {
			processErr := this.processKey(taskKey)
			var pbResult = &pb.UpdateHTTPCacheTaskKeysStatusRequest_KeyResult{
				Id:            taskKey.Id,
				NodeClusterId: taskKey.NodeClusterId,
				Error:         "",
			}

			if processErr != nil {
				pbResult.Error = processErr.Error()
			}

			taskGroup.Lock()
			pbResults = append(pbResults, pbResult)
			taskGroup.Unlock()
		})
	}

	taskGroup.Wait()

	_, err = rpcClient.HTTPCacheTaskKeyRPC.UpdateHTTPCacheTaskKeysStatus(rpcClient.Context(), &pb.UpdateHTTPCacheTaskKeysStatusRequest{KeyResults: pbResults})
	if err != nil {
		return err
	}

	return nil
}

func (this *HTTPCacheTaskManager) PushTaskKeys(keys []string) {
	select {
	case this.taskQueue <- &pb.PurgeServerCacheRequest{
		Keys:     keys,
		Prefixes: nil,
	}:
	default:
	}
}

func (this *HTTPCacheTaskManager) processKey(key *pb.HTTPCacheTaskKey) error {
	switch key.Type {
	case "purge":
		var storages = caches.SharedManager.FindAllStorages()
		for _, storage := range storages {
			switch key.KeyType {
			case "key":
				var cacheKeys = []string{key.Key}
				if strings.HasPrefix(key.Key, "http://") {
					cacheKeys = append(cacheKeys, strings.Replace(key.Key, "http://", "https://", 1))
				} else if strings.HasPrefix(key.Key, "https://") {
					cacheKeys = append(cacheKeys, strings.Replace(key.Key, "https://", "http://", 1))
				}

				// TODO 提升效率
				for _, cacheKey := range cacheKeys {
					var subKeys = []string{
						cacheKey,
						cacheKey + caches.SuffixMethod + "HEAD",
						cacheKey + caches.SuffixWebP,
						cacheKey + caches.SuffixPartial,
					}
					// TODO 根据实际缓存的内容进行组合
					for _, encoding := range compressions.AllEncodings() {
						subKeys = append(subKeys, cacheKey+caches.SuffixCompression+encoding)
						subKeys = append(subKeys, cacheKey+caches.SuffixWebP+caches.SuffixCompression+encoding)
					}

					err := storage.Purge(subKeys, "file")
					if err != nil {
						return err
					}
				}
			case "prefix":
				var prefixes = []string{key.Key}
				if strings.HasPrefix(key.Key, "http://") {
					prefixes = append(prefixes, strings.Replace(key.Key, "http://", "https://", 1))
				} else if strings.HasPrefix(key.Key, "https://") {
					prefixes = append(prefixes, strings.Replace(key.Key, "https://", "http://", 1))
				}

				err := storage.Purge(prefixes, "dir")
				if err != nil {
					return err
				}
			}
		}
	case "fetch":
		err := this.fetchKey(key)
		if err != nil {
			return err
		}
	default:
		return errors.New("invalid operation type '" + key.Type + "'")
	}

	return nil
}

// TODO 增加失败重试
func (this *HTTPCacheTaskManager) fetchKey(key *pb.HTTPCacheTaskKey) error {
	var fullKey = key.Key
	if !this.protocolReg.MatchString(fullKey) {
		fullKey = "https://" + fullKey
	}

	req, err := http.NewRequest(http.MethodGet, fullKey, nil)
	if err != nil {
		return fmt.Errorf("invalid url: '%s': %w", fullKey, err)
	}

	// TODO 可以在管理界面自定义Header
	req.Header.Set("X-Edge-Cache-Action", "fetch")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/85.0.4183.121 Safari/537.36") // TODO 可以定义
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	resp, err := this.httpClient().Do(req)
	if err != nil {
		err = this.simplifyErr(err)
		return fmt.Errorf("request failed: '%s': %w", fullKey, err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	// 处理502
	if resp.StatusCode == http.StatusBadGateway {
		return errors.New("read origin site timeout")
	}

	// 读取内容，以便于生成缓存
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		if err != io.EOF {
			err = this.simplifyErr(err)
			return fmt.Errorf("request failed: '%s': %w", fullKey, err)
		} else {
			err = nil
		}
	}

	return nil
}

func (this *HTTPCacheTaskManager) simplifyErr(err error) error {
	if err == nil {
		return nil
	}
	if os.IsTimeout(err) {
		return errors.New("timeout to read origin site")
	}

	return err
}

func (this *HTTPCacheTaskManager) httpClient() *http.Client {
	var timeout = serverconfigs.DefaultHTTPCachePolicyFetchTimeout

	var nodeConfig = sharedNodeConfig // copy
	if nodeConfig != nil {
		var cachePolicies = nodeConfig.HTTPCachePolicies // copy
		if len(cachePolicies) > 0 && cachePolicies[0].FetchTimeout != nil && cachePolicies[0].FetchTimeout.Count > 0 {
			var fetchTimeout = cachePolicies[0].FetchTimeout.Duration()
			if fetchTimeout > 0 {
				timeout = fetchTimeout
			}
		}
	}

	this.locker.Lock()
	defer this.locker.Unlock()

	client, ok := this.timeoutClientMap[timeout]
	if ok {
		return client
	}

	client = &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				_, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				conn, err := net.Dial(network, "127.0.0.1:"+port)
				if err != nil {
					return nil, err
				}

				return connutils.NewNoStat(conn), nil
			},
			MaxIdleConns:          128,
			MaxIdleConnsPerHost:   32,
			MaxConnsPerHost:       32,
			IdleConnTimeout:       2 * time.Minute,
			ExpectContinueTimeout: 1 * time.Second,
			TLSHandshakeTimeout:   0,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	this.timeoutClientMap[timeout] = client

	return client
}

// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package nodes

import (
	"context"
	"crypto/tls"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/caches"
	"github.com/TeaOSLab/EdgeNode/internal/compressions"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/iwind/TeaGo/Tea"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"
)

func init() {
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
	httpClient  *http.Client
	protocolReg *regexp.Regexp

	taskQueue chan *pb.PurgeServerCacheRequest
}

func NewHTTPCacheTaskManager() *HTTPCacheTaskManager {
	var duration = 30 * time.Second
	if Tea.IsTesting() {
		duration = 10 * time.Second
	}
	return &HTTPCacheTaskManager{
		ticker: time.NewTicker(duration),
		httpClient: &http.Client{
			Timeout: 10 * time.Minute, // TODO 可以设置请求超时时间
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					_, port, err := net.SplitHostPort(addr)
					if err != nil {
						return nil, err
					}
					return net.Dial(network, "127.0.0.1:"+port)
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
		},
		protocolReg: regexp.MustCompile(`^(?i)(http|https)://`),
		taskQueue:   make(chan *pb.PurgeServerCacheRequest, 1024),
	}
}

func (this *HTTPCacheTaskManager) Start() {
	// task queue
	goman.New(func() {
		rpcClient, _ := rpc.SharedRPC()

		if rpcClient != nil {
			for taskReq := range this.taskQueue {
				_, err := rpcClient.ServerRPC().PurgeServerCache(rpcClient.Context(), taskReq)
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

	resp, err := rpcClient.HTTPCacheTaskKeyRPC().FindDoingHTTPCacheTaskKeys(rpcClient.Context(), &pb.FindDoingHTTPCacheTaskKeysRequest{})
	if err != nil {
		return err
	}

	var keys = resp.HttpCacheTaskKeys
	if len(keys) == 0 {
		return nil
	}

	var pbResults = []*pb.UpdateHTTPCacheTaskKeysStatusRequest_KeyResult{}

	for _, key := range keys {
		err = this.processKey(key)

		var pbResult = &pb.UpdateHTTPCacheTaskKeysStatusRequest_KeyResult{
			Id:            key.Id,
			NodeClusterId: key.NodeClusterId,
			Error:         "",
		}

		if err != nil {
			pbResult.Error = err.Error()
		}
		pbResults = append(pbResults, pbResult)
	}

	_, err = rpcClient.HTTPCacheTaskKeyRPC().UpdateHTTPCacheTaskKeysStatus(rpcClient.Context(), &pb.UpdateHTTPCacheTaskKeysStatusRequest{KeyResults: pbResults})
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
// TODO 使用并发操作
func (this *HTTPCacheTaskManager) fetchKey(key *pb.HTTPCacheTaskKey) error {
	var fullKey = key.Key
	if !this.protocolReg.MatchString(fullKey) {
		fullKey = "https://" + fullKey
	}

	req, err := http.NewRequest(http.MethodGet, fullKey, nil)
	if err != nil {
		return errors.New("invalid url: " + fullKey + ": " + err.Error())
	}

	// TODO 可以在管理界面自定义Header
	req.Header.Set("X-Edge-Cache-Action", "fetch")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/85.0.4183.121 Safari/537.36") // TODO 可以定义
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	resp, err := this.httpClient.Do(req)
	if err != nil {
		return errors.New("request failed: " + fullKey + ": " + err.Error())
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	// 读取内容，以便于生成缓存
	_, _ = io.Copy(ioutil.Discard, resp.Body)

	// 处理502
	if resp.StatusCode == http.StatusBadGateway {
		return errors.New("read origin site timeout")
	}

	return nil
}

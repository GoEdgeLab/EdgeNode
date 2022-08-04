package utils

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"
)

// HTTP请求客户端管理
var timeoutClientMap = map[time.Duration]*http.Client{} // timeout => Client
var timeoutClientLocker = sync.Mutex{}

// DumpResponse 导出响应
func DumpResponse(resp *http.Response) (header []byte, body []byte, err error) {
	header, err = httputil.DumpResponse(resp, false)
	if err != nil {
		return
	}
	body, err = io.ReadAll(resp.Body)
	return
}

// NewHTTPClient 获取一个新的Client
func NewHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:          4096,
			MaxIdleConnsPerHost:   32,
			MaxConnsPerHost:       32,
			IdleConnTimeout:       2 * time.Minute,
			ExpectContinueTimeout: 1 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
}

// SharedHttpClient 获取一个公用的Client
func SharedHttpClient(timeout time.Duration) *http.Client {
	timeoutClientLocker.Lock()
	defer timeoutClientLocker.Unlock()

	client, ok := timeoutClientMap[timeout]
	if ok {
		return client
	}
	client = NewHTTPClient(timeout)
	timeoutClientMap[timeout] = client
	return client
}

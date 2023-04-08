package nodes

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"net/http"
)

// HTTPClient HTTP客户端
type HTTPClient struct {
	rawClient *http.Client
	accessAt  int64
}

// NewHTTPClient 获取新客户端对象
func NewHTTPClient(rawClient *http.Client) *HTTPClient {
	return &HTTPClient{
		rawClient: rawClient,
		accessAt:  fasttime.Now().Unix(),
	}
}

// RawClient 获取原始客户端对象
func (this *HTTPClient) RawClient() *http.Client {
	return this.rawClient
}

// UpdateAccessTime 更新访问时间
func (this *HTTPClient) UpdateAccessTime() {
	this.accessAt = fasttime.Now().Unix()
}

// AccessTime 获取访问时间
func (this *HTTPClient) AccessTime() int64 {
	return this.accessAt
}

// Close 关闭
func (this *HTTPClient) Close() {
	this.rawClient.CloseIdleConnections()
}

package nodes

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"net/http"
)

// HTTP客户端
type HTTPClient struct {
	rawClient *http.Client
	accessAt  int64
}

// 获取新客户端对象
func NewHTTPClient(rawClient *http.Client) *HTTPClient {
	return &HTTPClient{
		rawClient: rawClient,
		accessAt:  utils.UnixTime(),
	}
}

// 获取原始客户端对象
func (this *HTTPClient) RawClient() *http.Client {
	return this.rawClient
}

// 更新访问时间
func (this *HTTPClient) UpdateAccessTime() {
	this.accessAt = utils.UnixTime()
}

// 获取访问时间
func (this *HTTPClient) AccessTime() int64 {
	return this.accessAt
}

// 关闭
func (this *HTTPClient) Close() {
	this.rawClient.CloseIdleConnections()
}

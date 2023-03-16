package requests

import (
	"net/http"
)

type Request interface {
	// WAFRaw 原始请求
	WAFRaw() *http.Request

	// WAFRemoteIP 客户端IP
	WAFRemoteIP() string

	// WAFGetCacheBody 获取缓存中的Body
	WAFGetCacheBody() []byte

	// WAFSetCacheBody 设置Body
	WAFSetCacheBody(body []byte)

	// WAFReadBody 读取Body
	WAFReadBody(max int64) (data []byte, err error)

	// WAFRestoreBody 恢复Body
	WAFRestoreBody(data []byte)

	// WAFServerId 服务ID
	WAFServerId() int64

	// WAFClose 关闭当前请求所在的连接
	WAFClose()

	// WAFOnAction 动作回调
	WAFOnAction(action interface{}) (goNext bool)

	// WAFFingerprint 读取连接指纹
	WAFFingerprint() []byte

	// Format 格式化变量
	Format(string) string

	// DisableAccessLog 在当前请求中不使用访问日志
	DisableAccessLog()
}

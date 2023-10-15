package teaconst

const (
	Version = "1.2.10"

	ProductName = "Edge Node"
	ProcessName = "edge-node"

	Role = "node"

	EncryptKey    = "8f983f4d69b83aaa0d74b21a212f6967"
	EncryptMethod = "aes-256-cfb"

	// SystemdServiceName systemd
	SystemdServiceName = "edge-node"

	AccessLogSockName    = "edge-node.accesslog"
	CacheGarbageSockName = "edge-node.cache.garbage"
)

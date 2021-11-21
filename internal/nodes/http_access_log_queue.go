package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"strconv"
	"time"
)

var sharedHTTPAccessLogQueue = NewHTTPAccessLogQueue()

// HTTPAccessLogQueue HTTP访问日志队列
type HTTPAccessLogQueue struct {
	queue chan *pb.HTTPAccessLog

	rpcClient *rpc.RPCClient
}

// NewHTTPAccessLogQueue 获取新对象
func NewHTTPAccessLogQueue() *HTTPAccessLogQueue {
	// 队列中最大的值，超出此数量的访问日志会被丢弃
	// TODO 需要可以在界面中设置
	maxSize := 20000
	queue := &HTTPAccessLogQueue{
		queue: make(chan *pb.HTTPAccessLog, maxSize),
	}
	go queue.Start()

	return queue
}

// Start 开始处理访问日志
func (this *HTTPAccessLogQueue) Start() {
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		err := this.loop()
		if err != nil {
			remotelogs.Error("ACCESS_LOG_QUEUE", err.Error())
		}
	}
}

// Push 加入新访问日志
func (this *HTTPAccessLogQueue) Push(accessLog *pb.HTTPAccessLog) {
	select {
	case this.queue <- accessLog:
	default:

	}
}

// 上传访问日志
func (this *HTTPAccessLogQueue) loop() error {
	var accessLogs = []*pb.HTTPAccessLog{}
	var count = 0
	var timestamp int64
	var requestId = 1_000_000

Loop:
	for {
		select {
		case accessLog := <-this.queue:
			var unixTime = utils.UnixTime()
			if unixTime > timestamp {
				requestId = 1_000_000
				timestamp = unixTime
			} else {
				requestId++
			}

			// timestamp + requestId + nodeId
			accessLog.RequestId = strconv.FormatInt(unixTime, 10) + strconv.Itoa(requestId) + strconv.FormatInt(accessLog.NodeId, 10)

			accessLogs = append(accessLogs, accessLog)
			count++

			// 每次只提交 N 条访问日志，防止网络拥堵
			if count > 2000 {
				break Loop
			}
		default:
			break Loop
		}
	}

	if len(accessLogs) == 0 {
		return nil
	}

	// 发送到API
	if this.rpcClient == nil {
		client, err := rpc.SharedRPC()
		if err != nil {
			return err
		}
		this.rpcClient = client
	}

	_, err := this.rpcClient.HTTPAccessLogRPC().CreateHTTPAccessLogs(this.rpcClient.Context(), &pb.CreateHTTPAccessLogsRequest{HttpAccessLogs: accessLogs})
	if err != nil {
		return err
	}

	return nil
}

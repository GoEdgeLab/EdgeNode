package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"time"
)

var sharedHTTPAccessLogQueue = NewHTTPAccessLogQueue()

// HTTP访问日志队列
type HTTPAccessLogQueue struct {
	queue chan *pb.HTTPAccessLog
}

// 获取新对象
func NewHTTPAccessLogQueue() *HTTPAccessLogQueue {
	// 队列中最大的值，超出此数量的访问日志会被抛弃
	// TODO 需要可以在界面中设置
	maxSize := 10000
	queue := &HTTPAccessLogQueue{
		queue: make(chan *pb.HTTPAccessLog, maxSize),
	}
	go queue.Start()

	return queue
}

// 开始处理访问日志
func (this *HTTPAccessLogQueue) Start() {
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		err := this.loop()
		if err != nil {
			remotelogs.Error("ACCESS_LOG_QUEUE", err.Error())
		}
	}
}

// 加入新访问日志
func (this *HTTPAccessLogQueue) Push(accessLog *pb.HTTPAccessLog) {
	select {
	case this.queue <- accessLog:
	default:

	}
}

// 上传访问日志
func (this *HTTPAccessLogQueue) loop() error {
	accessLogs := []*pb.HTTPAccessLog{}
	count := 0
Loop:
	for {
		select {
		case accessLog := <-this.queue:
			accessLogs = append(accessLogs, accessLog)
			count++

			// 每次只提交 N 条访问日志，防止网络拥堵
			if count > 1000 {
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
	client, err := rpc.SharedRPC()
	if err != nil {
		return err
	}

	_, err = client.HTTPAccessLogRPC().CreateHTTPAccessLogs(client.Context(), &pb.CreateHTTPAccessLogsRequest{HttpAccessLogs: accessLogs})
	if err != nil {
		return err
	}

	return nil
}

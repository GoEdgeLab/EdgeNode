package nodes

import (
	"bytes"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"strings"
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
	goman.New(func() {
		queue.Start()
	})

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

Loop:
	for {
		select {
		case accessLog := <-this.queue:
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
		// 是否包含了invalid UTF-8
		if strings.Contains(err.Error(), "string field contains invalid UTF-8") {
			for _, accessLog := range accessLogs {
				this.toValidUTF8(accessLog)
			}

			// 重新提交
			_, err = this.rpcClient.HTTPAccessLogRPC().CreateHTTPAccessLogs(this.rpcClient.Context(), &pb.CreateHTTPAccessLogsRequest{HttpAccessLogs: accessLogs})
			return err
		}

		return err
	}

	return nil
}

func (this *HTTPAccessLogQueue) toValidUTF8(accessLog *pb.HTTPAccessLog) {
	accessLog.RemoteUser = this.toValidUTF8string(accessLog.RemoteUser)
	accessLog.RequestURI = this.toValidUTF8string(accessLog.RequestURI)
	accessLog.RequestPath = this.toValidUTF8string(accessLog.RequestPath)
	accessLog.RequestFilename = this.toValidUTF8string(accessLog.RequestFilename)
	accessLog.RequestBody = bytes.ToValidUTF8(accessLog.RequestBody, []byte{})

	for _, v := range accessLog.SentHeader {
		for index, s := range v.Values {
			v.Values[index] = this.toValidUTF8string(s)
		}
	}

	accessLog.Referer = this.toValidUTF8string(accessLog.Referer)
	accessLog.UserAgent = this.toValidUTF8string(accessLog.UserAgent)
	accessLog.Request = this.toValidUTF8string(accessLog.Request)
	accessLog.ContentType = this.toValidUTF8string(accessLog.ContentType)

	for k, c := range accessLog.Cookie {
		accessLog.Cookie[k] = this.toValidUTF8string(c)
	}

	accessLog.Args = this.toValidUTF8string(accessLog.Args)
	accessLog.QueryString = this.toValidUTF8string(accessLog.QueryString)

	for _, v := range accessLog.Header {
		for index, s := range v.Values {
			v.Values[index] = this.toValidUTF8string(s)
		}
	}
}

func (this *HTTPAccessLogQueue) toValidUTF8string(v string) string {
	return strings.ToValidUTF8(v, "")
}

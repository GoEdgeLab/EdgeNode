package nodes

import (
	"bytes"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"time"
	"unicode/utf8"
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
	var maxSize = 2_000 * (1 + utils.SystemMemoryGB()/2)
	if maxSize > 20_000 {
		maxSize = 20_000
	}

	var queue = &HTTPAccessLogQueue{
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
			if rpc.IsConnError(err) {
				remotelogs.Debug("ACCESS_LOG_QUEUE", err.Error())
			} else {
				remotelogs.Error("ACCESS_LOG_QUEUE", err.Error())
			}
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

	// 发送到本地
	if sharedHTTPAccessLogViewer.HasConns() {
		for _, accessLog := range accessLogs {
			sharedHTTPAccessLogViewer.Send(accessLog)
		}
	}

	// 发送到API
	if this.rpcClient == nil {
		client, err := rpc.SharedRPC()
		if err != nil {
			return err
		}
		this.rpcClient = client
	}

	_, err := this.rpcClient.HTTPAccessLogRPC.CreateHTTPAccessLogs(this.rpcClient.Context(), &pb.CreateHTTPAccessLogsRequest{HttpAccessLogs: accessLogs})
	if err != nil {
		// 是否包含了invalid UTF-8
		if strings.Contains(err.Error(), "string field contains invalid UTF-8") {
			for _, accessLog := range accessLogs {
				this.ToValidUTF8(accessLog)
			}

			// 重新提交
			_, err = this.rpcClient.HTTPAccessLogRPC.CreateHTTPAccessLogs(this.rpcClient.Context(), &pb.CreateHTTPAccessLogsRequest{HttpAccessLogs: accessLogs})
			return err
		}

		// 是否请求内容过大
		statusCode, ok := status.FromError(err)
		if ok && statusCode.Code() == codes.ResourceExhausted {
			// 去除Body
			for _, accessLog := range accessLogs {
				accessLog.RequestBody = nil
			}

			// 重新提交
			_, err = this.rpcClient.HTTPAccessLogRPC.CreateHTTPAccessLogs(this.rpcClient.Context(), &pb.CreateHTTPAccessLogsRequest{HttpAccessLogs: accessLogs})
			return err
		}

		return err
	}

	return nil
}

// ToValidUTF8 处理访问日志中的非UTF-8字节
func (this *HTTPAccessLogQueue) ToValidUTF8(accessLog *pb.HTTPAccessLog) {
	accessLog.RemoteAddr = utils.ToValidUTF8string(accessLog.RemoteAddr)
	accessLog.RemoteUser = utils.ToValidUTF8string(accessLog.RemoteUser)
	accessLog.RequestURI = utils.ToValidUTF8string(accessLog.RequestURI)
	accessLog.RequestPath = utils.ToValidUTF8string(accessLog.RequestPath)
	accessLog.RequestFilename = utils.ToValidUTF8string(accessLog.RequestFilename)
	accessLog.RequestBody = bytes.ToValidUTF8(accessLog.RequestBody, []byte{})
	accessLog.Host = utils.ToValidUTF8string(accessLog.Host)
	accessLog.Hostname = utils.ToValidUTF8string(accessLog.Hostname)

	for k, v := range accessLog.SentHeader {
		if !utf8.ValidString(k) {
			delete(accessLog.SentHeader, k)
			continue
		}

		for index, s := range v.Values {
			v.Values[index] = utils.ToValidUTF8string(s)
		}
	}

	accessLog.Referer = utils.ToValidUTF8string(accessLog.Referer)
	accessLog.UserAgent = utils.ToValidUTF8string(accessLog.UserAgent)
	accessLog.Request = utils.ToValidUTF8string(accessLog.Request)
	accessLog.ContentType = utils.ToValidUTF8string(accessLog.ContentType)

	for k, c := range accessLog.Cookie {
		if !utf8.ValidString(k) {
			delete(accessLog.Cookie, k)
			continue
		}
		accessLog.Cookie[k] = utils.ToValidUTF8string(c)
	}

	accessLog.Args = utils.ToValidUTF8string(accessLog.Args)
	accessLog.QueryString = utils.ToValidUTF8string(accessLog.QueryString)

	for k, v := range accessLog.Header {
		if !utf8.ValidString(k) {
			delete(accessLog.Header, k)
			continue
		}
		for index, s := range v.Values {
			v.Values[index] = utils.ToValidUTF8string(s)
		}
	}

	for k, v := range accessLog.Errors {
		accessLog.Errors[k] = utils.ToValidUTF8string(v)
	}
}

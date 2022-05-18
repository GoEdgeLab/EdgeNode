// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package nodes

import (
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/iwind/TeaGo/types"
	"net"
	"os"
	"sync"
	"sync/atomic"
)

var sharedHTTPAccessLogViewer = NewHTTPAccessLogViewer()

// HTTPAccessLogViewer 本地访问日志浏览器
type HTTPAccessLogViewer struct {
	sockFile string

	listener net.Listener
	connMap  map[int64]net.Conn // connId => net.Conn
	connId   int64
	locker   sync.Mutex
}

// NewHTTPAccessLogViewer 获取新对象
func NewHTTPAccessLogViewer() *HTTPAccessLogViewer {
	return &HTTPAccessLogViewer{
		sockFile: os.TempDir() + "/" + teaconst.AccessLogSockName,
		connMap:  map[int64]net.Conn{},
	}
}

// Start 启动
func (this *HTTPAccessLogViewer) Start() error {
	this.locker.Lock()
	defer this.locker.Unlock()

	if this.listener == nil {
		// remove if exists
		_ = os.Remove(this.sockFile)

		// start listening
		listener, err := net.Listen("unix", this.sockFile)
		if err != nil {
			return err
		}
		this.listener = listener

		go func() {
			for {
				conn, err := this.listener.Accept()
				if err != nil {
					remotelogs.Error("ACCESS_LOG", "start local reading failed: "+err.Error())
					break
				}

				this.locker.Lock()
				var connId = this.nextConnId()
				this.connMap[connId] = conn
				go func() {
					this.startReading(conn, connId)
				}()
				this.locker.Unlock()
			}
		}()
	}

	return nil
}

// HasConns 检查是否有连接
func (this *HTTPAccessLogViewer) HasConns() bool {
	this.locker.Lock()
	defer this.locker.Unlock()
	return len(this.connMap) > 0
}

// Send 发送日志
func (this *HTTPAccessLogViewer) Send(accessLog *pb.HTTPAccessLog) {
	var conns = []net.Conn{}
	this.locker.Lock()
	for _, conn := range this.connMap {
		conns = append(conns, conn)
	}
	this.locker.Unlock()

	if len(conns) == 0 {
		return
	}

	for _, conn := range conns {
		// ignore error
		_, _ = conn.Write([]byte(accessLog.RemoteAddr + " [" + accessLog.TimeLocal + "] \"" + accessLog.RequestMethod + " " + accessLog.Scheme + "://" + accessLog.Host + accessLog.RequestURI + " " + accessLog.Proto + "\" " + types.String(accessLog.Status) + " - " + fmt.Sprintf("%.2fms", accessLog.RequestTime*1000) + "\n"))
	}
}

func (this *HTTPAccessLogViewer) nextConnId() int64 {
	return atomic.AddInt64(&this.connId, 1)
}

func (this *HTTPAccessLogViewer) startReading(conn net.Conn, connId int64) {
	var buf = make([]byte, 1024)
	for {
		_, err := conn.Read(buf)
		if err != nil {
			this.locker.Lock()
			delete(this.connMap, connId)
			this.locker.Unlock()
			break
		}
	}
}

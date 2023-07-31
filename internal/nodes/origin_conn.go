// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package nodes

import (
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/TeaOSLab/EdgeNode/internal/zero"
	"net"
	"sync"
	"time"
)

const originConnCloseDelaySeconds = 3

var closingOriginConnMap = map[*OriginConn]zero.Zero{}
var closingOriginConnLocker = &sync.RWMutex{}

func init() {
	if !teaconst.IsMain {
		return
	}

	goman.New(func() {
		var ticker = time.NewTicker(originConnCloseDelaySeconds * time.Second)
		for range ticker.C {
			CleanOriginConnsTask()
		}
	})
}

func CleanOriginConnsTask() {
	var closingConns = []*OriginConn{}

	closingOriginConnLocker.RLock()
	for conn := range closingOriginConnMap {
		if conn.IsExpired() {
			closingConns = append(closingConns, conn)
		}
	}
	closingOriginConnLocker.RUnlock()

	if len(closingConns) > 0 {
		for _, conn := range closingConns {
			_ = conn.ForceClose()
			closingOriginConnLocker.Lock()
			delete(closingOriginConnMap, conn)
			closingOriginConnLocker.Unlock()
		}
	}
}

// OriginConn connection with origin site
type OriginConn struct {
	net.Conn

	lastReadOk bool
	lastReadAt int64
	isClosed   bool
}

// NewOriginConn create new origin connection
func NewOriginConn(rawConn net.Conn) net.Conn {
	return &OriginConn{Conn: rawConn}
}

// Read implement Read() for net.Conn interface
func (this *OriginConn) Read(b []byte) (n int, err error) {
	n, err = this.Conn.Read(b)
	this.lastReadOk = err == nil
	if this.lastReadOk {
		this.lastReadAt = fasttime.Now().Unix()
	}
	return
}

// Close implement Close() for net.Conn interface
func (this *OriginConn) Close() error {
	if this.lastReadOk && fasttime.Now().Unix()-this.lastReadAt <= originConnCloseDelaySeconds {
		closingOriginConnLocker.Lock()
		closingOriginConnMap[this] = zero.Zero{}
		closingOriginConnLocker.Unlock()
		return nil
	}

	this.isClosed = true
	return this.Conn.Close()
}

func (this *OriginConn) ForceClose() error {
	if this.isClosed {
		return nil
	}

	this.isClosed = true
	return this.Conn.Close()
}

func (this *OriginConn) IsExpired() bool {
	return fasttime.Now().Unix()-this.lastReadAt > originConnCloseDelaySeconds
}

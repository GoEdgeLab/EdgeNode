package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"net"
)

type UnixListener struct {
	BaseListener

	Listener net.Listener
}

func (this *UnixListener) Serve() error {
	// TODO
	return nil
}

func (this *UnixListener) Close() error {
	// TODO
	return nil
}

func (this *UnixListener) Reload(group *serverconfigs.ServerGroup) {
	this.Group = group
	this.Reset()
}

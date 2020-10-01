package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"net"
)

type UDPListener struct {
	BaseListener

	Listener net.Listener
}

func (this *UDPListener) Serve() error {
	// TODO
	return nil
}

func (this *UDPListener) Close() error {
	// TODO
	return nil
}

func (this *UDPListener) Reload(group *serverconfigs.ServerGroup) {
	this.Group = group
	this.Reset()
}

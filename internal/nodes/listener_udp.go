package nodes

import (
	"github.com/TeaOSLab/EdgeNode/internal/configs/serverconfigs"
	"net"
)

type UDPListener struct {
	BaseListener

	Group    *serverconfigs.ServerGroup
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

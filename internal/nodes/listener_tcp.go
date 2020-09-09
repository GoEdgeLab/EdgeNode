package nodes

import (
	"github.com/TeaOSLab/EdgeNode/internal/configs/serverconfigs"
	"net"
)

type TCPListener struct {
	BaseListener

	Group    *serverconfigs.ServerGroup
	Listener net.Listener
}

func (this *TCPListener) Serve() error {
	// TODO
	return nil
}

func (this *TCPListener) Close() error {
	// TODO
	return nil
}

package nodes

import (
	"context"
	"errors"
	"github.com/TeaOSLab/EdgeNode/internal/configs"
	"github.com/iwind/TeaGo/logs"
	"net"
	"net/http"
	"sync"
)

type Listener struct {
	group       *configs.ServerGroup
	isListening bool
	listener    interface{} // 监听器

	locker sync.RWMutex
}

func NewListener() *Listener {
	return &Listener{}
}

func (this *Listener) Reload(group *configs.ServerGroup) {
	this.locker.Lock()
	defer this.locker.Unlock()
	this.group = group
}

func (this *Listener) FullAddr() string {
	if this.group != nil {
		return this.group.FullAddr()
	}
	return ""
}

func (this *Listener) Listen() error {
	if this.group == nil {
		return nil
	}
	protocol := this.group.Protocol()
	switch protocol {
	case configs.ProtocolHTTP, configs.ProtocolHTTP4, configs.ProtocolHTTP6:
		return this.listenHTTP()
	case configs.ProtocolHTTPS, configs.ProtocolHTTPS4, configs.ProtocolHTTPS6:
		return this.ListenHTTPS()
	case configs.ProtocolTCP, configs.ProtocolTCP4, configs.ProtocolTCP6:
		return this.listenTCP()
	case configs.ProtocolTLS, configs.ProtocolTLS4, configs.ProtocolTLS6:
		return this.listenTLS()
	case configs.ProtocolUnix:
		return this.listenUnix()
	case configs.ProtocolUDP:
		return this.listenUDP()
	default:
		return errors.New("unknown protocol '" + protocol + "'")
	}
}

func (this *Listener) Close() error {
	// TODO 需要实现
	return nil
}

func (this *Listener) listenHTTP() error {
	listener, err := this.createListener()
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte("Hello, World"))
	})
	server := &http.Server{
		Addr:    this.group.Addr(),
		Handler: mux,
	}

	go func() {
		err = server.Serve(listener)
		if err != nil {
			logs.Println("[LISTENER]" + err.Error())
		}
	}()
	return nil
}

func (this *Listener) ListenHTTPS() error {
	// TODO 需要实现
	return nil
}

func (this *Listener) listenTCP() error {
	// TODO 需要实现
	return nil
}

func (this *Listener) listenTLS() error {
	// TODO 需要实现
	return nil
}

func (this *Listener) listenUnix() error {
	// TODO 需要实现
	return nil
}

func (this *Listener) listenUDP() error {
	// TODO 需要实现
	return nil
}

func (this *Listener) createListener() (net.Listener, error) {
	listenConfig := net.ListenConfig{
		Control:   nil,
		KeepAlive: 0,
	}

	switch this.group.Protocol() {
	case configs.ProtocolHTTP4, configs.ProtocolHTTPS4, configs.ProtocolTLS4:
		return listenConfig.Listen(context.Background(), "tcp4", this.group.Addr())
	case configs.ProtocolHTTP6, configs.ProtocolHTTPS6, configs.ProtocolTLS6:
		return listenConfig.Listen(context.Background(), "tcp6", this.group.Addr())
	}

	return listenConfig.Listen(context.Background(), "tcp", this.group.Addr())
}

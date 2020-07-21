package nodes

import (
	"errors"
	"github.com/TeaOSLab/EdgeNode/internal/configs"
	"sync"
)

type Listener struct {
	group  *configs.ServerGroup
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
	case configs.ProtocolHTTP:
		return this.listenHTTP()
	case configs.ProtocolHTTPS:
		return this.ListenHTTPS()
	case configs.ProtocolTCP:
		return this.listenTCP()
	case configs.ProtocolTLS:
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
	return nil
}

func (this *Listener) ListenHTTPS() error {
	return nil
}

func (this *Listener) listenTCP() error {
	return nil
}

func (this *Listener) listenTLS() error {
	return nil
}

func (this *Listener) listenUnix() error {
	return nil
}

func (this *Listener) listenUDP() error {
	return nil
}

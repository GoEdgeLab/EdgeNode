package configs

import "testing"

func TestServerConfig_Protocols(t *testing.T) {
	{
		server := NewServerConfig()
		t.Log(server.FullAddresses())
	}

	{
		server := NewServerConfig()
		server.HTTP = &HTTPProtocolConfig{IsOn: true, Listen: []string{"127.0.0.1:1234"}}
		server.HTTPS = &HTTPSProtocolConfig{IsOn: true, Listen: []string{"127.0.0.1:1234"}}
		server.TCP = &TCPProtocolConfig{IsOn: true, Listen: []string{"127.0.0.1:1234"}}
		server.TLS = &TLSProtocolConfig{IsOn: true, Listen: []string{"127.0.0.1:1234"}}
		server.Unix = &UnixProtocolConfig{IsOn: true, Listen: []string{"127.0.0.1:1234"}}
		server.UDP = &UDPProtocolConfig{IsOn: true, Listen: []string{"127.0.0.1:1234"}}
		t.Log(server.FullAddresses())
	}
}

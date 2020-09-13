package configs

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	_ "github.com/iwind/TeaGo/bootstrap"
	"github.com/iwind/TeaGo/logs"
	"testing"
)

func TestSharedNodeConfig(t *testing.T) {
	{
		config, err := SharedNodeConfig()
		if err != nil {
			t.Fatal(err)
		}
		t.Log(config)
	}

	// read from memory cache
	{
		config, err := SharedNodeConfig()
		if err != nil {
			t.Fatal(err)
		}
		t.Log(config)
	}
}

func TestNodeConfig_Groups(t *testing.T) {
	config := &NodeConfig{}
	config.Servers = []*serverconfigs.ServerConfig{
		{
			IsOn: true,
			HTTP: &serverconfigs.HTTPProtocolConfig{
				BaseProtocol: serverconfigs.BaseProtocol{
					IsOn: true,
					Listen: []*serverconfigs.NetworkAddressConfig{
						{
							Protocol:  serverconfigs.ProtocolHTTP,
							Host:      "127.0.0.1",
							PortRange: "1234",
						},
						{
							Protocol:  serverconfigs.ProtocolHTTP,
							PortRange: "8080",
						},
					},
				},
			},
		},
		{
			HTTP: &serverconfigs.HTTPProtocolConfig{
				BaseProtocol: serverconfigs.BaseProtocol{
					IsOn: true,
					Listen: []*serverconfigs.NetworkAddressConfig{
						{
							Protocol:  serverconfigs.ProtocolHTTP,
							PortRange: "8080",
						},
					},
				},
			},
		},
	}
	logs.PrintAsJSON(config.AvailableGroups(), t)
}

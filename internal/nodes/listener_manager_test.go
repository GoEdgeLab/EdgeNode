package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"testing"
)

func TestListenerManager_Listen(t *testing.T) {
	manager := NewListenerManager()
	err := manager.Start(&nodeconfigs.NodeConfig{
		Servers: []*serverconfigs.ServerConfig{
			{
				IsOn: true,
				HTTP: &serverconfigs.HTTPProtocolConfig{
					BaseProtocol: serverconfigs.BaseProtocol{
						IsOn: true,
						Listen: []*serverconfigs.NetworkAddressConfig{
							{
								Protocol:  serverconfigs.ProtocolHTTP,
								PortRange: "1234",
							},
						},
					},
				},
			},
			{
				IsOn: true,
				HTTP: &serverconfigs.HTTPProtocolConfig{
					BaseProtocol: serverconfigs.BaseProtocol{
						IsOn: true,
						Listen: []*serverconfigs.NetworkAddressConfig{
							{
								Protocol:  serverconfigs.ProtocolHTTP,
								PortRange: "1235",
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	err = manager.Start(&nodeconfigs.NodeConfig{
		Servers: []*serverconfigs.ServerConfig{
			{
				IsOn: true,
				HTTP: &serverconfigs.HTTPProtocolConfig{
					BaseProtocol: serverconfigs.BaseProtocol{
						IsOn: true,
						Listen: []*serverconfigs.NetworkAddressConfig{
							{
								Protocol:  serverconfigs.ProtocolHTTP,
								PortRange: "1234",
							},
						},
					},
				},
			},
			{
				IsOn: true,
				HTTP: &serverconfigs.HTTPProtocolConfig{
					BaseProtocol: serverconfigs.BaseProtocol{
						IsOn: true,
						Listen: []*serverconfigs.NetworkAddressConfig{
							{
								Protocol:  serverconfigs.ProtocolHTTP,
								PortRange: "1236",
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Log("all ok")
}

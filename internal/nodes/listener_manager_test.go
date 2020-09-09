package nodes

import (
	"github.com/TeaOSLab/EdgeNode/internal/configs"
	"testing"
)

func TestListenerManager_Listen(t *testing.T) {
	manager := NewListenerManager()
	err := manager.Start(&configs.NodeConfig{
		Servers: []*configs.ServerConfig{
			{
				IsOn: true,
				HTTP: &configs.HTTPProtocolConfig{
					BaseProtocol: configs.BaseProtocol{
						IsOn: true,
						Listen: []*configs.NetworkAddressConfig{
							{
								Protocol:  configs.ProtocolHTTP,
								PortRange: "1234",
							},
						},
					},
				},
			},
			{
				IsOn: true,
				HTTP: &configs.HTTPProtocolConfig{
					BaseProtocol: configs.BaseProtocol{
						IsOn: true,
						Listen: []*configs.NetworkAddressConfig{
							{
								Protocol:  configs.ProtocolHTTP,
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

	err = manager.Start(&configs.NodeConfig{
		Servers: []*configs.ServerConfig{
			{
				IsOn: true,
				HTTP: &configs.HTTPProtocolConfig{
					BaseProtocol: configs.BaseProtocol{
						IsOn: true,
						Listen: []*configs.NetworkAddressConfig{
							{
								Protocol:  configs.ProtocolHTTP,
								PortRange: "1234",
							},
						},
					},
				},
			},
			{
				IsOn: true,
				HTTP: &configs.HTTPProtocolConfig{
					BaseProtocol: configs.BaseProtocol{
						IsOn: true,
						Listen: []*configs.NetworkAddressConfig{
							{
								Protocol:  configs.ProtocolHTTP,
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

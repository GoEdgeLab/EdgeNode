package configs

import (
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
	config.Servers = []*ServerConfig{
		{
			IsOn: true,
			HTTP: &HTTPProtocolConfig{
				IsOn:   true,
				Listen: []string{"127.0.0.1:1234", ":8080"},
			},
		},
		{
			HTTP: &HTTPProtocolConfig{
				IsOn:   true,
				Listen: []string{":8080"},
			},
		},
	}
	logs.PrintAsJSON(config.AvailableGroups(), t)
}

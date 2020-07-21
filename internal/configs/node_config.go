package configs

import (
	"github.com/go-yaml/yaml"
	"github.com/iwind/TeaGo/Tea"
	"io/ioutil"
)

var sharedNodeConfig *NodeConfig = nil

type NodeConfig struct {
	Id      string          `yaml:"id"`
	Servers []*ServerConfig `yaml:"servers"`
}

func SharedNodeConfig() (*NodeConfig, error) {
	sharedLocker.Lock()
	defer sharedLocker.Unlock()

	if sharedNodeConfig != nil {
		return sharedNodeConfig, nil
	}

	data, err := ioutil.ReadFile(Tea.ConfigFile("node.yaml"))
	if err != nil {
		return &NodeConfig{}, err
	}

	config := &NodeConfig{}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return &NodeConfig{}, err
	}

	sharedNodeConfig = config
	return config, nil
}

// 根据网络地址和协议分组
func (this *NodeConfig) AvailableGroups() []*ServerGroup {
	groupMapping := map[string]*ServerGroup{} // protocol://addr => Server Group
	for _, server := range this.Servers {
		if !server.IsOn {
			continue
		}
		for _, addr := range server.FullAddresses() {
			group, ok := groupMapping[addr]
			if ok {
				group.Add(server)
			} else {
				group = NewServerGroup(addr)
				group.Add(server)
			}
			groupMapping[addr] = group
		}
	}
	result := []*ServerGroup{}
	for _, group := range groupMapping {
		result = append(result, group)
	}
	return result
}

package configs

import (
	"github.com/TeaOSLab/EdgeNode/internal/configs/serverconfigs"
	"github.com/go-yaml/yaml"
	"github.com/iwind/TeaGo/Tea"
	"io/ioutil"
)

var sharedNodeConfig *NodeConfig = nil

type NodeConfig struct {
	Id      string                        `yaml:"id" json:"id"`
	IsOn    bool                          `yaml:"isOn" json:"isOn"`
	Servers []*serverconfigs.ServerConfig `yaml:"servers" json:"servers"`
	Version int                           `yaml:"version" json:"version"`
}

// 取得当前节点配置单例
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

// 刷新当前节点配置
func ReloadNodeConfig() error {
	sharedLocker.Lock()
	sharedNodeConfig = nil
	sharedLocker.Unlock()

	_, err := SharedNodeConfig()
	return err
}

// 根据网络地址和协议分组
func (this *NodeConfig) AvailableGroups() []*serverconfigs.ServerGroup {
	groupMapping := map[string]*serverconfigs.ServerGroup{} // protocol://addr => Server Group
	for _, server := range this.Servers {
		if !server.IsOn {
			continue
		}
		for _, addr := range server.FullAddresses() {
			group, ok := groupMapping[addr]
			if ok {
				group.Add(server)
			} else {
				group = serverconfigs.NewServerGroup(addr)
				group.Add(server)
			}
			groupMapping[addr] = group
		}
	}
	result := []*serverconfigs.ServerGroup{}
	for _, group := range groupMapping {
		result = append(result, group)
	}
	return result
}

func (this *NodeConfig) Init() error {
	for _, server := range this.Servers {
		err := server.Init()
		if err != nil {
			return err
		}
	}

	return nil
}

// 写入到文件
func (this *NodeConfig) Save() error {
	sharedLocker.Lock()
	defer sharedLocker.Unlock()

	data, err := yaml.Marshal(this)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(Tea.ConfigFile("node.yaml"), data, 0777)
}

// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package firewalls

// MockFirewall 模拟防火墙
type MockFirewall struct {
}

func NewMockFirewall() *MockFirewall {
	return &MockFirewall{}
}

// Name 名称
func (this *MockFirewall) Name() string {
	return "mock"
}

// IsReady 是否已准备被调用
func (this *MockFirewall) IsReady() bool {
	return true
}

// IsMock 是否为模拟
func (this *MockFirewall) IsMock() bool {
	return true
}

// AllowPort 允许端口
func (this *MockFirewall) AllowPort(port int, protocol string) error {
	_ = port
	_ = protocol
	return nil
}

// RemovePort 删除端口
func (this *MockFirewall) RemovePort(port int, protocol string) error {
	_ = port
	_ = protocol
	return nil
}

// RejectSourceIP 拒绝某个源IP连接
func (this *MockFirewall) RejectSourceIP(ip string, timeoutSeconds int) error {
	_ = ip
	_ = timeoutSeconds
	return nil
}

// DropSourceIP 丢弃某个源IP数据
func (this *MockFirewall) DropSourceIP(ip string, timeoutSeconds int, async bool) error {
	_ = ip
	_ = timeoutSeconds
	return nil
}

// RemoveSourceIP 删除某个源IP
func (this *MockFirewall) RemoveSourceIP(ip string) error {
	_ = ip
	return nil
}

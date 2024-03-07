// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package firewalls

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
	"io"
	"net/http"
	"net/url"
)

type HTTPFirewall struct {
	client   *http.Client
	endpoint string
}

func NewHTTPFirewall(endpoint string) *HTTPFirewall {
	return &HTTPFirewall{
		client:   http.DefaultClient,
		endpoint: endpoint,
	}
}

// Name 名称
func (this *HTTPFirewall) Name() string {
	result, err := this.get("/name", nil)
	if err != nil {
		return ""
	}
	return result.GetString("name")
}

// IsReady 是否已准备被调用
func (this *HTTPFirewall) IsReady() bool {
	result, err := this.get("/isReady", nil)
	if err != nil {
		return false
	}
	return result.GetBool("result")
}

// IsMock 是否为模拟
func (this *HTTPFirewall) IsMock() bool {
	result, err := this.get("/isMock", nil)
	if err != nil {
		return false
	}
	return result.GetBool("result")
}

// AllowPort 允许端口
func (this *HTTPFirewall) AllowPort(port int, protocol string) error {
	_, err := this.get("/allowPort", map[string]string{
		"port":     types.String(port),
		"protocol": protocol,
	})
	return err
}

// RemovePort 删除端口
func (this *HTTPFirewall) RemovePort(port int, protocol string) error {
	_, err := this.get("/removePort", map[string]string{
		"port":     types.String(port),
		"protocol": protocol,
	})
	return err
}

// RejectSourceIP 拒绝某个源IP连接
func (this *HTTPFirewall) RejectSourceIP(ip string, timeoutSeconds int) error {
	_, err := this.get("/rejectSourceIP", map[string]string{
		"ip":             ip,
		"timeoutSeconds": types.String(timeoutSeconds),
	})
	return err
}

// DropSourceIP 丢弃某个源IP数据
// ip 要封禁的IP
// timeoutSeconds 过期时间
// async 是否异步
func (this *HTTPFirewall) DropSourceIP(ip string, timeoutSeconds int, async bool) error {
	var asyncString = "false"
	if async {
		asyncString = "true"
	}
	_, err := this.get("/dropSourceIP", map[string]string{
		"ip":             ip,
		"timeoutSeconds": types.String(timeoutSeconds),
		"async":          asyncString,
	})
	return err
}

// RemoveSourceIP 删除某个源IP
func (this *HTTPFirewall) RemoveSourceIP(ip string) error {
	_, err := this.get("/removeSourceIP", map[string]string{
		"ip": ip,
	})
	return err
}

func (this *HTTPFirewall) get(path string, args map[string]string) (result maps.Map, err error) {
	var urlString = this.endpoint + path
	if len(args) > 0 {
		var query = &url.Values{}
		for k, v := range args {
			query.Add(k, v)
		}
		urlString += "?" + query.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, urlString, nil)
	if err != nil {
		return nil, err
	}

	resp, err := this.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("server response code '" + types.String(resp.StatusCode) + "'")
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	if len(data) == 0 {
		return maps.Map{}, nil
	}

	result = maps.Map{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return
}

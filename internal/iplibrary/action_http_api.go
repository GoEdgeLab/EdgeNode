package iplibrary

import (
	"bytes"
	"encoding/json"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/iwind/TeaGo/maps"
	"net/http"
	"time"
)

var httpAPIClient = &http.Client{
	Timeout: 5 * time.Second,
}

type HTTPAPIAction struct {
	BaseAction

	config *firewallconfigs.FirewallActionHTTPAPIConfig
}

func NewHTTPAPIAction() *HTTPAPIAction {
	return &HTTPAPIAction{}
}

func (this *HTTPAPIAction) Init(config *firewallconfigs.FirewallActionConfig) error {
	this.config = &firewallconfigs.FirewallActionHTTPAPIConfig{}
	err := this.convertParams(config.Params, this.config)
	if err != nil {
		return err
	}

	if len(this.config.URL) == 0 {
		return NewFataError("'url' should not be empty")
	}

	return nil
}

func (this *HTTPAPIAction) AddItem(listType IPListType, item *pb.IPItem) error {
	return this.runAction("addItem", listType, item)
}

func (this *HTTPAPIAction) DeleteItem(listType IPListType, item *pb.IPItem) error {
	return this.runAction("deleteItem", listType, item)
}

func (this *HTTPAPIAction) runAction(action string, listType IPListType, item *pb.IPItem) error {
	if item == nil {
		return nil
	}

	// TODO 增加节点ID等信息
	m := maps.Map{
		"action":   action,
		"listType": listType,
		"item": maps.Map{
			"type":      item.Type,
			"ipFrom":    item.IpFrom,
			"ipTo":      item.IpTo,
			"expiredAt": item.ExpiredAt,
		},
	}
	mJSON, err := json.Marshal(m)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, this.config.URL, bytes.NewReader(mJSON))
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", teaconst.GlobalProductName+"-Node/"+teaconst.Version)
	resp, err := httpAPIClient.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	return nil
}

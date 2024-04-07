// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package waf

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"net/url"
)

type InfoArg struct {
	ActionId         int64  `json:"1,omitempty"`
	Timestamp        int64  `json:"2,omitempty"`
	URL              string `json:"3,omitempty"`
	PolicyId         int64  `json:"4,omitempty"`
	GroupId          int64  `json:"5,omitempty"`
	SetId            int64  `json:"6,omitempty"`
	UseLocalFirewall bool   `json:"7,omitempty"`
	Life             int32  `json:"8,omitempty"`
	Scope            string `json:"9,omitempty"`
	RemoteIP         string `json:"10,omitempty"`
}

func (this *InfoArg) IsValid() bool {
	return this.Timestamp > 0
}

func (this *InfoArg) Encode() (string, error) {
	if this.Timestamp <= 0 {
		this.Timestamp = fasttime.Now().Unix()
	}

	return utils.SimpleEncryptObject(this)
}

func (this *InfoArg) URLEncoded() (string, error) {
	encodedString, err := this.Encode()
	if err != nil {
		return "", err
	}
	return url.QueryEscape(encodedString), nil
}

func (this *InfoArg) Decode(encodedString string) error {
	return utils.SimpleDecryptObjet(encodedString, this)
}

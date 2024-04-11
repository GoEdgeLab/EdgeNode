// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package waf

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"github.com/TeaOSLab/EdgeNode/internal/utils"
)

type AllowCookieInfo struct {
	SetId     int64
	ExpiresAt int64
}

func (this *AllowCookieInfo) Encode() (string, error) {
	if this.SetId < 0 {
		this.SetId = 0
	}
	if this.ExpiresAt < 0 {
		this.ExpiresAt = 0
	}

	var result = make([]byte, 16)
	binary.BigEndian.PutUint64(result, uint64(this.SetId))
	binary.BigEndian.PutUint64(result[8:], uint64(this.ExpiresAt))
	return base64.StdEncoding.EncodeToString(utils.SimpleEncrypt(result)), nil
}

func (this *AllowCookieInfo) Decode(encodedString string) error {
	data, err := base64.StdEncoding.DecodeString(encodedString)
	if err != nil {
		return err
	}

	var result = utils.SimpleDecrypt(data)
	if len(result) != 16 {
		return errors.New("unexpected data length")
	}

	this.SetId = int64(binary.BigEndian.Uint64(result[:8]))
	this.ExpiresAt = int64(binary.BigEndian.Uint64(result[8:16]))

	return nil
}

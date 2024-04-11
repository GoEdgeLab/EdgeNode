// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package waf_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/fasttime"
	"github.com/TeaOSLab/EdgeNode/internal/waf"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/types"
	"testing"
)

func TestAllowCookieInfo_Encode(t *testing.T) {
	var a = assert.NewAssertion(t)

	var info = &waf.AllowCookieInfo{
		SetId:     123,
		ExpiresAt: fasttime.Now().Unix(),
	}
	data, err := info.Encode()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("encrypted: ["+types.String(len(data))+"]", data)

	var info2 = &waf.AllowCookieInfo{}
	err = info2.Decode(data)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%+v", info2)
	a.IsTrue(info.SetId == info2.SetId)
	a.IsTrue(info.ExpiresAt == info2.ExpiresAt)
}

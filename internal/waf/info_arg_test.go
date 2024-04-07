// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package waf_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf"
	"github.com/iwind/TeaGo/types"
	"testing"
	"time"
)

func TestInfoArg_Encode(t *testing.T) {
	var info = &waf.InfoArg{
		ActionId:         1,
		Timestamp:        time.Now().Unix(),
		URL:              "https://example.com/hello",
		PolicyId:         2,
		GroupId:          3,
		SetId:            4,
		UseLocalFirewall: true,
		Scope:            "global",
	}

	encodedString, err := info.Encode()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("["+types.String(len(encodedString))+"]", encodedString)

	{
		urlEncodedString, encodeErr := info.URLEncoded()
		if encodeErr != nil {
			t.Fatal(encodeErr)
		}
		t.Log("["+types.String(len(urlEncodedString))+"]", urlEncodedString)
	}

	var info2 = &waf.InfoArg{}
	err = info2.Decode(encodedString)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", info2)
}

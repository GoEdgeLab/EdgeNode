// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package js

import (
	"net/url"
	"testing"
)

func TestURL(t *testing.T) {
	raw, err := url.Parse("https://iwind:123456@goedge.cn/docs?name=Libai&age=20#a=b")
	if err != nil {
		t.Fatal(err)
	}
	var u = NewURL(raw)
	t.Log("host:", u.Host())
	t.Log("hash:", u.Hash())
}

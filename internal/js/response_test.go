// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package js_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/js"
	"testing"
)

func TestNewResponse(t *testing.T) {
	var resp = js.NewResponse(&testRequest{})
	resp.AddHeader("Vary", "1.0")
	resp.AddHeader("Vary", "2.0")
	resp.SetHeader("Server", "edgejs/1.0")
	t.Logf("%#v", resp.Header())
}

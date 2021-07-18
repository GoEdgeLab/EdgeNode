// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package jsonutils

import (
	"encoding/json"
	"testing"
)

func PrintT(obj interface{}, t *testing.T) {
	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		t.Log(err)
	} else {
		t.Log(string(data))
	}
}

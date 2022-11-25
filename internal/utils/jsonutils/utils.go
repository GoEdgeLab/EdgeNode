// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package jsonutils

import (
	"bytes"
	"encoding/json"
	"testing"
)

func PrintT(obj any, t *testing.T) {
	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		t.Log(err)
	} else {
		t.Log(string(data))
	}
}

func Equal(obj1 any, obj2 any) bool {
	data1, err := json.Marshal(obj1)
	if err != nil {
		return false
	}

	data2, err := json.Marshal(obj2)
	if err != nil {
		return false
	}

	return bytes.Equal(data1, data2)
}

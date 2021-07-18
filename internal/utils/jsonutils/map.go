// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package jsonutils

import (
	"encoding/json"
	"github.com/iwind/TeaGo/maps"
)

func MapToObject(m maps.Map, ptr interface{}) error {
	if m == nil {
		return nil
	}
	mJSON, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(mJSON, ptr)
}

func ObjectToMap(ptr interface{}) (maps.Map, error) {
	if ptr == nil {
		return maps.Map{}, nil
	}
	ptrJSON, err := json.Marshal(ptr)
	if err != nil {
		return nil, err
	}
	var result = maps.Map{}
	err = json.Unmarshal(ptrJSON, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

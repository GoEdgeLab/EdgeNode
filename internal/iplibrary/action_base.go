package iplibrary

import (
	"encoding/json"
	"github.com/iwind/TeaGo/maps"
)

type BaseAction struct {
}

func (this *BaseAction) Close() error {
	return nil
}

func (this *BaseAction) convertParams(params maps.Map, ptr interface{}) error {
	data, err := json.Marshal(params)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, ptr)
	if err != nil {
		return err
	}
	return nil
}

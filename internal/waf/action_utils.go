package waf

import (
	"encoding/json"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/iwind/TeaGo/maps"
	"reflect"
	"sync/atomic"
)

var seedActionId int64 = 1

func FindActionInstance(action ActionString, options maps.Map) ActionInterface {
	for _, def := range AllActions {
		if def.Code == action {
			if def.Type != nil {
				// create new instance
				var ptrValue = reflect.New(def.Type)
				var instance = ptrValue.Interface().(ActionInterface)
				instance.SetActionId(atomic.AddInt64(&seedActionId, 1))

				if len(options) > 0 {
					optionsJSON, err := json.Marshal(options)
					if err != nil {
						remotelogs.Error("WAF_FindActionInstance", "encode options to json failed: "+err.Error())
					} else {
						err = json.Unmarshal(optionsJSON, instance)
						if err != nil {
							remotelogs.Error("WAF_FindActionInstance", "decode options from json failed: "+err.Error())
						}
					}
				}

				return instance
			}

			// return shared instance
			return def.Instance
		}
	}
	return nil
}

func FindActionName(action ActionString) string {
	for _, def := range AllActions {
		if def.Code == action {
			return def.Name
		}
	}
	return ""
}

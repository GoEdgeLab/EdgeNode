// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package js

import (
	"encoding/json"
	"github.com/iwind/TeaGo/logs"
	"reflect"
)

type Console struct {
}

func (this *Console) Log(args ...interface{}) {
	for index, arg := range args {
		if arg != nil {
			switch arg.(type) {
			case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, string:
			default:
				var argType = reflect.TypeOf(arg)

				// 是否有String()方法，如果有直接调用
				method, ok := argType.MethodByName("String")
				if ok && method.Type.NumIn() == 1 && method.Type.NumOut() == 1 && method.Type.Out(0).Kind() == reflect.String {
					args[index] = method.Func.Call([]reflect.Value{reflect.ValueOf(arg)})[0].String()
					continue
				}

				// 转为JSON
				argJSON, err := this.toJSON(arg)
				if err != nil {
					if argType.Kind() == reflect.Func {
						args[index] = "[function]"
					} else {
						args[index] = "[object]"
					}
				} else {
					args[index] = string(argJSON)
				}
			}
		} else {
			args[index] = "null"
		}
	}
	logs.Println(append([]interface{}{"[js][console]"}, args...)...)
}

func (this *Console) toJSON(o interface{}) ([]byte, error) {
	return json.Marshal(o)
}

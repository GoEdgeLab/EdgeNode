// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package js

import (
	"errors"
	"github.com/dop251/goja"
	"github.com/iwind/TeaGo/logs"
	"reflect"
	"strings"
)

var sharedPrograms []*goja.Program
var sharedConsole = &Console{}

func init() {
	// compile programs
}

type VM struct {
	vm *goja.Runtime
}

func NewVM() *VM {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	// programs
	for _, program := range sharedPrograms {
		_, _ = vm.RunProgram(program)
	}

	v := &VM{vm: vm}
	v.initVM()
	return v
}

func (this *VM) Set(name string, obj interface{}) error {
	return this.vm.Set(name, obj)
}

func (this *VM) AddConstructor(name string, instance interface{}) error {
	objType := reflect.TypeOf(instance)

	if objType.Kind() != reflect.Ptr {
		return errors.New("instance should be pointer")
	}

	// construct
	newMethod, ok := objType.MethodByName("JSNew")
	if !ok {
		return errors.New("can not find 'JSNew()' method in '" + objType.Elem().Name() + "'")
	}

	var err = this.Set(name, func(call goja.ConstructorCall) *goja.Object {
		if newMethod.Type.NumIn() != 2 {
			this.throw(errors.New(objType.Elem().Name() + ".JSNew() should accept a '[]goja.Value' argument"))
			return nil
		}
		if newMethod.Type.In(1).String() != "[]goja.Value" {
			this.throw(errors.New(objType.Elem().Name() + ".JSNew() should accept a '[]goja.Value' argument"))
			return nil
		}

		// new
		var results = newMethod.Func.Call([]reflect.Value{reflect.ValueOf(instance), reflect.ValueOf(call.Arguments)})
		if len(results) == 0 {
			this.throw(errors.New(objType.Elem().Name() + ".JSNew() should return a valid instance"))
			return nil
		}
		var result = results[0]
		if result.Type() != objType {
			this.throw(errors.New(objType.Elem().Name() + ".JSNew() should return a same instance"))
			return nil
		}

		// methods
		var resultType = result.Type()
		var numMethod = result.NumMethod()
		for i := 0; i < numMethod; i++ {
			var method = resultType.Method(i)
			var methodName = strings.ToLower(method.Name[:1]) + method.Name[1:]
			err := call.This.Set(methodName, result.MethodByName(method.Name).Interface())
			if err != nil {
				this.throw(err)
				continue
			}
		}

		//  支持属性
		var numField = result.Elem().Type().NumField()
		for i := 0; i < numField; i++ {
			var field = result.Elem().Field(i)
			if !field.CanInterface() {
				continue
			}
			var fieldType = objType.Elem().Field(i)
			tag, ok := fieldType.Tag.Lookup("json")
			if !ok {
				tag = fieldType.Name
				tag = strings.ToLower(tag[:1]) + tag[1:]
			} else {
				// TODO 校验tag是否符合变量语法
			}
			err := call.This.Set(tag, field.Interface())
			if err != nil {
				this.throw(err)
				continue
			}
		}

		return nil
	})
	return err
}

func (this *VM) RunString(str string) (goja.Value, error) {
	defer func() {
		e := recover()
		if e != nil {
			// TODO 需要打印trace
			logs.Println("panic:", e)
		}
	}()
	return this.vm.RunString(str)
}

func (this *VM) SetRequest(req RequestInterface) {
	{
		err := this.vm.Set("http", NewHTTP(req))
		if err != nil {
			this.throw(err)
		}
	}
}

func (this *VM) initVM() {
	{
		err := this.vm.Set("console", sharedConsole)
		if err != nil {
			this.throw(err)
		}
	}
}

func (this *VM) throw(err error) {
	if err == nil {
		return
	}

	// TODO
	logs.Println("js:VM:error: " + err.Error())
}

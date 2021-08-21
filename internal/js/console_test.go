// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package js

import (
	"testing"
)

func TestConsole_Log(t *testing.T) {
	{
		vm := NewVM()
		_, err := vm.RunString("console.log('Hello', 'world')")
		if err != nil {
			t.Fatal(err)
		}
	}
	{
		vm := NewVM()
		_, err := vm.RunString("console.log(null, true, false, 10, 10.123)")
		if err != nil {
			t.Fatal(err)
		}
	}
	{
		vm := NewVM()
		_, err := vm.RunString("console.log({ a:1, b:2 })")
		if err != nil {
			t.Fatal(err)
		}
	}
	{
		vm := NewVM()
		_, err := vm.RunString("console.log(console.log)")
		if err != nil {
			t.Fatal(err)
		}
	}
}

// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package js

import (
	"github.com/dop251/goja"
	"testing"
	"time"
)

func TestNewVM(t *testing.T) {
	before := time.Now()
	defer func() {
		t.Log(time.Since(before).Seconds()*1000, "ms")
	}()

	vm := NewVM()
	{
		v, err := vm.RunString("JSON.stringify({\"a\":\"b\"})")
		if err != nil {
			t.Fatal(err)
		}
		t.Log("JSON.stringify():", v)
	}
	{
		v, err := vm.RunString(`JSON.parse('{\"a\":\"b\"}')`)
		if err != nil {
			t.Fatal(err)
		}
		t.Log("JSON.parse():", v)
	}
	{
		err := vm.AddConstructor("Url", &URL{})
		if err != nil {
			t.Fatal("add constructor error:", err)
		}
		_, err = vm.RunString(`
{
	let u = new Url("https://goedge.cn/docs?v=1")
	console.log("host:", u.host(), u.uri())
}
{
	let u = new Url("https://teaos.cn/downloads?v=1")
	console.log("host:", u.host(), u.uri())
}

{
	let u = new Url()
	console.log("host:", u.host(), u.uri())
}

{
	let u = new Url("a", "b", "c")
	console.log("host:", u.host(), u.uri())
}
`)
		if err != nil {
			t.Fatal("add constructor error:" + err.Error())
		}
	}
}

func TestVM_Program(t *testing.T) {
	var s = `
{
	let u = new Url("https://goedge.cn/docs?v=1")
	//console.log("host:", u.host(), u.uri())
}
{
	let u = new Url("https://teaos.cn/downloads?v=1")
	//console.log("host:", u.host(), u.uri())
}

{
	let u = new Url()
	//console.log("host:", u.host(), u.uri())
}

{
	let u = new Url("a", "b", "c")
	//console.log("host:", u.host(), u.uri())
}
`
	program := goja.MustCompile("s", s, true)

	before := time.Now()
	defer func() {
		t.Log(time.Since(before).Seconds()*1000, "ms")
	}()

	vm := NewVM()
	err := vm.AddConstructor("Url", &URL{})
	if err != nil {
		t.Fatal("add constructor error:", err)
	}
	//_, err = vm.RunString(s)
	_, err = vm.vm.RunProgram(program)
	if err != nil {
		t.Fatal("add constructor error:" + err.Error())
	}
}

func Benchmark_Program(b *testing.B) {
	var s = `
{
	let u = new Url("https://goedge.cn/docs?v=1")
	//console.log("host:", u.host(), u.uri())
}
{
	let u = new Url("https://teaos.cn/downloads?v=1")
	//console.log("host:", u.host(), u.uri())
}

{
	let u = new Url()
	//console.log("host:", u.host(), u.uri())
}

{
	let u = new Url("a", "b", "c")
	//console.log("host:", u.host(), u.uri())
}
{
	let u = new Url("https://goedge.cn/docs?v=1")
	//console.log("host:", u.host(), u.uri())
}
{
	let u = new Url("https://teaos.cn/downloads?v=1")
	//console.log("host:", u.host(), u.uri())
}

{
	let u = new Url()
	//console.log("host:", u.host(), u.uri())
}

{
	let u = new Url("a", "b", "c")
	//console.log("host:", u.host(), u.uri())
}
`
	program := goja.MustCompile("s", s, true)

	vm := NewVM()

	err := vm.AddConstructor("Url", &URL{})
	if err != nil {
		b.Fatal("add constructor error:", err)
	}

	for i := 0; i < b.N; i++ {
		//_, err = vm.RunString(s)
		_, err = vm.vm.RunProgram(program)
		if err != nil {
			b.Fatal("add constructor error:" + err.Error())
		}
	}
}

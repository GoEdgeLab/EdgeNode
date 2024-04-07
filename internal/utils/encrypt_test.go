// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package utils_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/maps"
	"sync"
	"testing"
)

func TestSimpleEncrypt(t *testing.T) {
	var a = assert.NewAssertion(t)

	var arr = []string{"Hello", "World", "People"}
	for _, s := range arr {
		var value = []byte(s)
		var encoded = utils.SimpleEncrypt(value)
		t.Log(encoded, string(encoded))
		var decoded = utils.SimpleDecrypt(encoded)
		t.Log(decoded, string(decoded))
		a.IsTrue(s == string(decoded))
	}
}

func TestSimpleEncryptObject(t *testing.T) {
	var a = assert.NewAssertion(t)

	type Obj struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	encoded, err := utils.SimpleEncryptObject(&Obj{Name: "lily", Age: 20})
	if err != nil {
		t.Fatal(err)
	}

	var obj = &Obj{}
	err = utils.SimpleDecryptObjet(encoded, obj)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%#v", obj)
	a.IsTrue(obj.Name == "lily")
	a.IsTrue(obj.Age == 20)
}

func TestSimpleEncrypt_Concurrent(t *testing.T) {
	var wg = sync.WaitGroup{}
	var arr = []string{"Hello", "World", "People"}
	wg.Add(len(arr))
	for _, s := range arr {
		go func(s string) {
			defer wg.Done()
			t.Log(string(utils.SimpleDecrypt(utils.SimpleEncrypt([]byte(s)))))
		}(s)
	}
	wg.Wait()
}

func TestSimpleEncryptMap(t *testing.T) {
	var m = maps.Map{
		"s": "Hello",
		"i": 20,
		"b": true,
	}
	encodedResult, err := utils.SimpleEncryptMap(m)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("result:", encodedResult)

	decodedResult, err := utils.SimpleDecryptMap(encodedResult)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(decodedResult)
}

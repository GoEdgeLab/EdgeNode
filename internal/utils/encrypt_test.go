// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package utils

import (
	"github.com/iwind/TeaGo/maps"
	"sync"
	"testing"
)

func TestSimpleEncrypt(t *testing.T) {
	var arr = []string{"Hello", "World", "People"}
	for _, s := range arr {
		var value = []byte(s)
		encoded := SimpleEncrypt(value)
		t.Log(encoded, string(encoded))
		decoded := SimpleDecrypt(encoded)
		t.Log(decoded, string(decoded))
	}
}

func TestSimpleEncrypt_Concurrent(t *testing.T) {
	wg := sync.WaitGroup{}
	var arr = []string{"Hello", "World", "People"}
	wg.Add(len(arr))
	for _, s := range arr {
		go func(s string) {
			defer wg.Done()
			t.Log(string(SimpleDecrypt(SimpleEncrypt([]byte(s)))))
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
	encodedResult, err := SimpleEncryptMap(m)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("result:", encodedResult)

	decodedResult, err := SimpleDecryptMap(encodedResult)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(decodedResult)
}

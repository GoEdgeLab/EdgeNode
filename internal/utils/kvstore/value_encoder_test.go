// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package kvstore_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/kvstore"
	"github.com/iwind/TeaGo/assert"
	"testing"
)

func TestStringValueEncoder_Encode(t *testing.T) {
	var a = assert.NewAssertion(t)

	var encoder = kvstore.NewStringValueEncoder[string]()
	data, err := encoder.Encode("abcdefg")
	if err != nil {
		t.Fatal(err)
	}

	value, err := encoder.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	a.IsTrue(value == "abcdefg")
}

func TestIntValueEncoder_Encode(t *testing.T) {
	var a = assert.NewAssertion(t)

	{
		var encoder = kvstore.NewIntValueEncoder[int8]()
		data, err := encoder.Encode(1)
		if err != nil {
			t.Fatal(err)
		}
		v, err := encoder.Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(v == 1)
		t.Log("int8", string(data), "=>", data, "=>", v)
	}

	{
		var encoder = kvstore.NewIntValueEncoder[int8]()
		data, err := encoder.Encode(-1)
		if err != nil {
			t.Fatal(err)
		}
		v, err := encoder.Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(v == -1)
		t.Log("int8", string(data), "=>", data, "=>", v)
	}

	{
		var encoder = kvstore.NewIntValueEncoder[int16]()
		data, err := encoder.Encode(123)
		if err != nil {
			t.Fatal(err)
		}
		v, err := encoder.Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(v == 123)
		t.Log("int16", string(data), "=>", data, "=>", v)
	}

	{
		var encoder = kvstore.NewIntValueEncoder[int16]()
		data, err := encoder.Encode(-123)
		if err != nil {
			t.Fatal(err)
		}
		v, err := encoder.Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(v == -123)
		t.Log("int16", string(data), "=>", data, "=>", v)
	}

	{
		var encoder = kvstore.NewIntValueEncoder[int32]()
		data, err := encoder.Encode(123)
		if err != nil {
			t.Fatal(err)
		}
		v, err := encoder.Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(v == 123)
		t.Log("int32", string(data), "=>", data, "=>", v)
	}

	{
		var encoder = kvstore.NewIntValueEncoder[int32]()
		data, err := encoder.Encode(-123)
		if err != nil {
			t.Fatal(err)
		}
		v, err := encoder.Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(v == -123)
		t.Log("int32", string(data), "=>", data, "=>", v)
	}

	{
		var encoder = kvstore.NewIntValueEncoder[int64]()
		data, err := encoder.Encode(123456)
		if err != nil {
			t.Fatal(err)
		}
		v, err := encoder.Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(v == 123456)
		t.Log("int64", string(data), "=>", data, "=>", v)
	}

	{
		var encoder = kvstore.NewIntValueEncoder[int64]()
		data, err := encoder.Encode(1234567890)
		if err != nil {
			t.Fatal(err)
		}
		v, err := encoder.Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(v == 1234567890)
		t.Log("int64", string(data), "=>", data, "=>", v)
	}

	{
		var encoder = kvstore.NewIntValueEncoder[int64]()
		data, err := encoder.Encode(-123456)
		if err != nil {
			t.Fatal(err)
		}
		v, err := encoder.Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(v == -123456)
		t.Log("int64", string(data), "=>", data, "=>", v)
	}

	{
		var encoder = kvstore.NewIntValueEncoder[int]()
		data, err := encoder.Encode(123)
		if err != nil {
			t.Fatal(err)
		}
		v, err := encoder.Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(v == 123)
		t.Log("int", string(data), "=>", data, "=>", v)
	}

	{
		var encoder = kvstore.NewIntValueEncoder[int]()
		data, err := encoder.Encode(-123)
		if err != nil {
			t.Fatal(err)
		}
		v, err := encoder.Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(v == -123)
		t.Log("int", string(data), "=>", data, "=>", v)
	}

	{
		var encoder = kvstore.NewIntValueEncoder[uint]()
		data, err := encoder.Encode(123)
		if err != nil {
			t.Fatal(err)
		}
		v, err := encoder.Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(v == 123)
		t.Log("uint", string(data), "=>", data, "=>", v)
	}

	{
		var encoder = kvstore.NewIntValueEncoder[uint8]()
		data, err := encoder.Encode(97)
		if err != nil {
			t.Fatal(err)
		}
		v, err := encoder.Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(v == 97)
		t.Log("uint8", string(data), "=>", data, "=>", v)
	}

	{
		var encoder = kvstore.NewIntValueEncoder[uint16]()
		data, err := encoder.Encode(123)
		if err != nil {
			t.Fatal(err)
		}
		v, err := encoder.Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(v == 123)
		t.Log("uint16", string(data), "=>", data, "=>", v)
	}

	{
		var encoder = kvstore.NewIntValueEncoder[uint32]()
		data, err := encoder.Encode(123)
		if err != nil {
			t.Fatal(err)
		}
		v, err := encoder.Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(v == 123)
		t.Log("uint32", string(data), "=>", data, "=>", v)
	}

	{
		var encoder = kvstore.NewIntValueEncoder[uint64]()
		data, err := encoder.Encode(123)
		if err != nil {
			t.Fatal(err)
		}
		v, err := encoder.Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(v == 123)
		t.Log("uint64", string(data), "=>", data, "=>", v)
	}

	{
		var encoder = kvstore.NewIntValueEncoder[uint64]()
		data, err := encoder.Encode(1234567890)
		if err != nil {
			t.Fatal(err)
		}
		v, err := encoder.Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		a.IsTrue(v == 1234567890)
		t.Log("uint64", string(data), "=>", data, "=>", v)
	}
}

func TestBytesValueEncoder_Encode(t *testing.T) {
	var encoder = kvstore.NewBytesValueEncoder[[]byte]()
	{
		data, err := encoder.Encode(nil)
		if err != nil {
			t.Fatal(err)
		}
		value, err := encoder.Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(data, "=>", value)
	}

	{
		data, err := encoder.Encode([]byte("ABC"))
		if err != nil {
			t.Fatal(err)
		}
		value, err := encoder.Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(data, "=>", value)
	}
}

func TestBytesValueEncoder_Bool(t *testing.T) {
	var encoder = kvstore.NewBoolValueEncoder[bool]()
	{
		data, err := encoder.Encode(true)
		if err != nil {
			t.Fatal(err)
		}
		value, err := encoder.Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(data, "=>", value)
	}

	{
		data, err := encoder.Encode(false)
		if err != nil {
			t.Fatal(err)
		}
		value, err := encoder.Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(data, "=>", value)
	}

	{
		value, err := encoder.Decode(nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Log("nil", "=>", value)
	}

	{
		value, err := encoder.Decode([]byte{1, 2, 3, 4})
		if err != nil {
			t.Fatal(err)
		}
		t.Log("nil", "=>", value)
	}
}

type objectType struct {
	Name string `json:"1"`
	Age  int    `json:"2"`
}

type objectTypeEncoder[T objectType] struct {
	kvstore.BaseObjectEncoder[T]
}

func (this *objectTypeEncoder[T]) EncodeField(value T, fieldName string) ([]byte, error) {
	return nil, nil
}

func TestBaseObjectEncoder_Encode(t *testing.T) {
	var encoder = &objectTypeEncoder[objectType]{}

	{
		data, err := encoder.Encode(objectType{
			Name: "lily",
			Age:  20,
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Log("encoded:", string(data))
	}

	{
		value, err := encoder.Decode([]byte(`{"1":"lily","2":20}`))
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("decoded: %+v", value)
	}
}

type objectType2 struct {
	Name string `json:"1"`
	Age  int    `json:"2"`
}

type objectTypeEncoder2[T interface{ *objectType2 }] struct {
	kvstore.BaseObjectEncoder[T]
}

func (this *objectTypeEncoder2[T]) EncodeField(value T, fieldName string) ([]byte, error) {
	switch fieldName {
	case "Name":
		return []byte(any(value).(*objectType2).Name), nil
	}
	return nil, nil
}

func TestBaseObjectEncoder_Encode2(t *testing.T) {
	var encoder = &objectTypeEncoder2[*objectType2]{}

	{
		data, err := encoder.Encode(&objectType2{
			Name: "lily",
			Age:  20,
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Log("encoded:", string(data))
	}

	{
		value, err := encoder.Decode([]byte(`{"1":"lily","2":20}`))
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("decoded: %+v", value)
	}

	{
		field, err := encoder.EncodeField(&objectType2{
			Name: "lily",
			Age:  20,
		}, "Name")
		if err != nil {
			t.Fatal(err)
		}
		t.Log("encoded field:", string(field))
	}
}

func BenchmarkStringValueEncoder_Encode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var encoder = kvstore.NewStringValueEncoder[string]()
		data, err := encoder.Encode("1234567890")
		if err != nil {
			b.Fatal(err)
		}
		v, err := encoder.Decode(data)
		if err != nil {
			b.Fatal(err)
		}
		_ = v
	}
}

func BenchmarkIntValueEncoder_Encode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var encoder = kvstore.NewIntValueEncoder[int64]()
		data, err := encoder.Encode(1234567890)
		if err != nil {
			b.Fatal(err)
		}
		v, err := encoder.Decode(data)
		if err != nil {
			b.Fatal(err)
		}
		_ = v
	}
}

func BenchmarkUIntValueEncoder_Encode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var encoder = kvstore.NewIntValueEncoder[uint64]()
		data, err := encoder.Encode(1234567890)
		if err != nil {
			b.Fatal(err)
		}
		v, err := encoder.Decode(data)
		if err != nil {
			b.Fatal(err)
		}
		_ = v
	}
}

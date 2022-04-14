package caches

import "testing"

func TestMemoryReader_Header(t *testing.T) {
	item := &MemoryItem{
		ExpiresAt:   0,
		HeaderValue: []byte("0123456789"),
		BodyValue:   nil,
		Status:      2000,
	}
	reader := NewMemoryReader(item)
	buf := make([]byte, 6)
	err := reader.ReadHeader(buf, func(n int) (goNext bool, err error) {
		t.Log("buf:", string(buf[:n]))
		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMemoryReader_Body(t *testing.T) {
	item := &MemoryItem{
		ExpiresAt:   0,
		HeaderValue: nil,
		BodyValue:   []byte("0123456789"),
		Status:      2000,
	}
	reader := NewMemoryReader(item)
	buf := make([]byte, 6)
	err := reader.ReadBody(buf, func(n int) (goNext bool, err error) {
		t.Log("buf:", string(buf[:n]))
		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMemoryReader_Body_Range(t *testing.T) {
	item := &MemoryItem{
		ExpiresAt:   0,
		HeaderValue: nil,
		BodyValue:   []byte("0123456789"),
		Status:      2000,
	}
	reader := NewMemoryReader(item)
	buf := make([]byte, 6)
	var err error
	{
		err = reader.ReadBodyRange(buf, 0, 0, func(n int) (goNext bool, err error) {
			t.Log("[0, 0]", "body:", string(buf[:n]))
			return true, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	{
		err = reader.ReadBodyRange(buf, 7, 7, func(n int) (goNext bool, err error) {
			t.Log("[7, 7]", "body:", string(buf[:n]))
			return true, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	{
		err = reader.ReadBodyRange(buf, 0, 10, func(n int) (goNext bool, err error) {
			t.Log("[0, 10]", "body:", string(buf[:n]))
			return true, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	{
		err = reader.ReadBodyRange(buf, 3, 5, func(n int) (goNext bool, err error) {
			t.Log("[3, 5]", "body:", string(buf[:n]))
			return true, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	{
		err = reader.ReadBodyRange(buf, -1, -3, func(n int) (goNext bool, err error) {
			t.Log("[, -3]", "body:", string(buf[:n]))
			return true, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	{
		err = reader.ReadBodyRange(buf, 3, -1, func(n int) (goNext bool, err error) {
			t.Log("[3, ]", "body:", string(buf[:n]))
			return true, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

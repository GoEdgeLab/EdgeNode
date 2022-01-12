package events_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"testing"
)

func TestOn(t *testing.T) {
	type User struct {
		name string
	}
	var u = &User{}
	var u2 = &User{}

	events.On("hello", func() {
		t.Log("world")
	})
	events.On("hello", func() {
		t.Log("world2")
	})
	events.OnKey("hello", u, func() {
		t.Log("world3")
	})
	events.OnKey("hello", u, func() {
		t.Log("world4")
	})
	events.Remove(u)
	events.Remove(u2)
	events.OnKey("hello2", nil, func() {
		t.Log("world2")
	})
	events.Notify("hello")
}

package nodes

import (
	"github.com/TeaOSLab/EdgeNode/internal/configs"
	"testing"
)

func TestListener_Listen(t *testing.T) {
	listener := NewListener()

	group := configs.NewServerGroup("http://:1234")

	listener.Reload(group)
	err := listener.Listen()
	if err != nil {
		t.Fatal(err)
	}
}

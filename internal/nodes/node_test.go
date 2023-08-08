package nodes

import (
	_ "github.com/iwind/TeaGo/bootstrap"
	"testing"
)

func TestNode_Start(t *testing.T) {
	node := NewNode()
	node.Start()
}

func TestNode_Test(t *testing.T) {
	node := NewNode()
	err := node.Test()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}

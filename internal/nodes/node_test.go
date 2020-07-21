package nodes

import (
	_ "github.com/iwind/TeaGo/bootstrap"
	"testing"
)

func TestNode(t *testing.T) {
	node := NewNode()
	node.Start()
}

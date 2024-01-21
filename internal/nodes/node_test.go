package nodes

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	_ "github.com/iwind/TeaGo/bootstrap"
	"testing"
)

func TestNode_Start(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var node = NewNode()
	node.Start()
}

func TestNode_Test(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var node = NewNode()
	err := node.Test()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}

package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/golang/protobuf/proto"
	_ "github.com/iwind/TeaGo/bootstrap"
	"io"
	"strconv"
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

func TestNode_Proto_Buffer(t *testing.T) {
	buff := proto.NewBuffer([]byte{})
	for i := 0; i < 10; i++ {
		err := buff.EncodeMessage(&pb.NodeStreamMessage{
			RequestId: int64(i),
			Code:      "msg" + strconv.Itoa(i),
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	for i := 0; i < 11; i++ {
		msg := &pb.NodeStreamMessage{}
		err := buff.DecodeMessage(msg)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			} else {
				t.Fatal(err)
			}
		}
		t.Log(msg.Code, msg.RequestId)
	}
}

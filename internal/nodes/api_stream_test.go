package nodes

import "testing"

func TestAPIStream_Start(t *testing.T) {
	apiStream := NewAPIStream()
	apiStream.Start()
}

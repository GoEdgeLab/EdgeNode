package shared

import (
	"fmt"
	"testing"
)

func TestHeaderList_FormatHeaders(t *testing.T) {
	list := &HeaderList{}

	for i := 0; i < 5; i++ {
		list.AddRequestHeader(&HeaderConfig{
			IsOn:  true,
			Name:  "A" + fmt.Sprintf("%d", i),
			Value: "ABCDEFGHIJ${name}KLM${hello}NEFGHIJILKKKk",
		})
	}

	err := list.Init()
	if err != nil {
		t.Fatal(err)
	}
}

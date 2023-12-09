package waf_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf"
	"github.com/iwind/TeaGo/assert"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/maps"
	"runtime"
	"testing"
)

func TestFindActionInstance(t *testing.T) {
	a := assert.NewAssertion(t)

	t.Logf("ActionBlock: %p", waf.FindActionInstance(waf.ActionBlock, nil))
	t.Logf("ActionBlock: %p", waf.FindActionInstance(waf.ActionBlock, nil))
	t.Logf("ActionGoGroup: %p", waf.FindActionInstance(waf.ActionGoGroup, nil))
	t.Logf("ActionGoGroup: %p", waf.FindActionInstance(waf.ActionGoGroup, nil))
	t.Logf("ActionGoSet: %p", waf.FindActionInstance(waf.ActionGoSet, nil))
	t.Logf("ActionGoSet: %p", waf.FindActionInstance(waf.ActionGoSet, nil))
	t.Logf("ActionGoSet: %#v", waf.FindActionInstance(waf.ActionGoSet, maps.Map{"groupId": "a", "setId": "b"}))

	a.IsTrue(waf.FindActionInstance(waf.ActionGoSet, nil) != waf.FindActionInstance(waf.ActionGoSet, nil))
}

func TestFindActionInstance_Options(t *testing.T) {
	//t.Logf("%p", FindActionInstance(ActionBlock, maps.Map{}))
	//t.Logf("%p", FindActionInstance(ActionBlock, maps.Map{}))
	//logs.PrintAsJSON(FindActionInstance(ActionBlock, maps.Map{}), t)
	logs.PrintAsJSON(waf.FindActionInstance(waf.ActionBlock, maps.Map{
		"timeout": 3600,
	}), t)
}

func BenchmarkFindActionInstance(b *testing.B) {
	runtime.GOMAXPROCS(1)
	for i := 0; i < b.N; i++ {
		waf.FindActionInstance(waf.ActionGoSet, nil)
	}
}

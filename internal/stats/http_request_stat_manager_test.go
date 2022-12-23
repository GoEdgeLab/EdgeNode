package stats

import (
	iplib "github.com/TeaOSLab/EdgeCommon/pkg/iplibrary"
	_ "github.com/iwind/TeaGo/bootstrap"
	"github.com/iwind/TeaGo/logs"
	"testing"
)

func TestHTTPRequestStatManager_Loop_Region(t *testing.T) {
	err := iplib.InitDefault()
	if err != nil {
		t.Fatal(err)
	}

	var manager = NewHTTPRequestStatManager()
	manager.AddRemoteAddr(11, "202.196.0.20", 0, false)
	manager.AddRemoteAddr(11, "202.196.0.20", 0, false) // 重复添加一个测试相加
	manager.AddRemoteAddr(11, "8.8.8.8", 0, false)

	/**for i := 0; i < 100; i++ {
		manager.AddRemoteAddr(11, strconv.Itoa(rands.Int(10, 250))+"."+strconv.Itoa(rands.Int(10, 250))+"."+strconv.Itoa(rands.Int(10, 250))+".8")
	}**/
	err = manager.Loop()
	if err != nil {
		t.Fatal(err)
	}
	logs.PrintAsJSON(manager.cityMap, t)
	logs.PrintAsJSON(manager.providerMap, t)

	err = manager.Upload()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}

func TestHTTPRequestStatManager_Loop_UserAgent(t *testing.T) {
	var manager = NewHTTPRequestStatManager()
	manager.AddUserAgent(1, "Mozilla/5.0 (Macintosh; Intel Mac OS X 11_1_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.96 Safari/537.36", "")
	manager.AddUserAgent(1, "Mozilla/5.0 (Macintosh; Intel Mac OS X 11_1_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.96 Safari/537.36", "")
	manager.AddUserAgent(1, "Mozilla/5.0 (Macintosh; Intel Mac OS X 11) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/76 Safari/537.36", "")
	manager.AddUserAgent(1, "Mozilla/5.0 (Windows NT 10.0; WOW64; rv:49.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.96 Safari/537.36", "")
	manager.AddUserAgent(1, "Mozilla/5.0 (Windows NT 6.1; WOW64; Trident/7.0; rv:11.0) like Gecko", "")
	err := manager.Loop()
	if err != nil {
		t.Fatal(err)
	}
	logs.PrintAsJSON(manager.systemMap, t)
	logs.PrintAsJSON(manager.browserMap, t)

	err = manager.Upload()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}

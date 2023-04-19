// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.
//go:build linux

package nftables_test

import (
	"errors"
	"github.com/TeaOSLab/EdgeNode/internal/firewalls/nftables"
	"github.com/iwind/TeaGo/types"
	"github.com/mdlayher/netlink"
	"net"
	"testing"
	"time"
)

func getIPv4Set(t *testing.T) *nftables.Set {
	var table = getIPv4Table(t)
	set, err := table.GetSet("test_ipv4_set")
	if err != nil {
		if err == nftables.ErrSetNotFound {
			set, err = table.AddSet("test_ipv4_set", &nftables.SetOptions{
				KeyType:    nftables.TypeIPAddr,
				HasTimeout: true,
			})
			if err != nil {
				t.Fatal(err)
			}
		} else {
			t.Fatal(err)
		}
	}
	return set
}

func TestSet_AddElement(t *testing.T) {
	var set = getIPv4Set(t)
	err := set.AddElement(net.ParseIP("192.168.2.31").To4(), &nftables.ElementOptions{Timeout: 86400 * time.Second}, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}

func TestSet_DeleteElement(t *testing.T) {
	var set = getIPv4Set(t)
	err := set.DeleteElement(net.ParseIP("192.168.2.31").To4())
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}

func TestSet_Batch(t *testing.T) {
	var batch = getIPv4Set(t).Batch()

	for _, ip := range []string{"192.168.2.30", "192.168.2.31", "192.168.2.32", "192.168.2.33", "192.168.2.34"} {
		var ipData = net.ParseIP(ip).To4()
		//err := batch.DeleteElement(ipData)
		//if err != nil {
		//	t.Fatal(err)
		//}
		err := batch.AddElement(ipData, &nftables.ElementOptions{Timeout: 10 * time.Second})
		if err != nil {
			t.Fatal(err)
		}
	}

	err := batch.Commit()
	if err != nil {
		t.Logf("%#v", errors.Unwrap(err).(*netlink.OpError))
		t.Fatal(err)
	}
	t.Log("ok")
}

func TestSet_Add_Many(t *testing.T) {
	var set = getIPv4Set(t)

	for i := 0; i < 255; i++ {
		t.Log(i)
		for j := 0; j < 255; j++ {
			var ip = "192.167." + types.String(i) + "." + types.String(j)
			var ipData = net.ParseIP(ip).To4()
			err := set.Batch().AddElement(ipData, &nftables.ElementOptions{Timeout: 3600 * time.Second})
			if err != nil {
				t.Fatal(err)
			}

			if j%10 == 0 {
				err = set.Batch().Commit()
				if err != nil {
					t.Fatal(err)
				}
			}
		}
		err := set.Batch().Commit()
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Log("ok")
}

/**func TestSet_Flush(t *testing.T) {
	var set = getIPv4Set(t)
	err := set.Flush()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}**/

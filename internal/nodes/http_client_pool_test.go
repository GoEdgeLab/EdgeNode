package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"runtime"
	"testing"
	"time"
)

func TestHTTPClientPool_Client(t *testing.T) {
	pool := NewHTTPClientPool()

	{
		origin := &serverconfigs.OriginConfig{
			Id:      1,
			Version: 2,
			Addr:    &serverconfigs.NetworkAddressConfig{Host: "127.0.0.1", PortRange: "1234"},
		}
		err := origin.Init()
		if err != nil {
			t.Fatal(err)
		}
		{
			client, err := pool.Client(nil, origin, origin.Addr.PickAddress())
			if err != nil {
				t.Fatal(err)
			}
			t.Log("client:", client)
		}
		for i := 0; i < 10; i++ {
			client, err := pool.Client(nil, origin, origin.Addr.PickAddress())
			if err != nil {
				t.Fatal(err)
			}
			t.Log("client:", client)
		}
	}
}

func TestHTTPClientPool_cleanClients(t *testing.T) {
	origin := &serverconfigs.OriginConfig{
		Id:      1,
		Version: 2,
		Addr:    &serverconfigs.NetworkAddressConfig{Host: "127.0.0.1", PortRange: "1234"},
	}
	err := origin.Init()
	if err != nil {
		t.Fatal(err)
	}

	pool := NewHTTPClientPool()
	pool.clientExpiredDuration = 2 * time.Second

	for i := 0; i < 10; i++ {
		t.Log("get", i)
		_, _ = pool.Client(nil, origin, origin.Addr.PickAddress())
		time.Sleep(1 * time.Second)
	}
}

func BenchmarkHTTPClientPool_Client(b *testing.B) {
	runtime.GOMAXPROCS(1)

	origin := &serverconfigs.OriginConfig{
		Id:      1,
		Version: 2,
		Addr:    &serverconfigs.NetworkAddressConfig{Host: "127.0.0.1", PortRange: "1234"},
	}
	err := origin.Init()
	if err != nil {
		b.Fatal(err)
	}

	pool := NewHTTPClientPool()
	for i := 0; i < b.N; i++ {
		_, _ = pool.Client(nil, origin, origin.Addr.PickAddress())
	}
}

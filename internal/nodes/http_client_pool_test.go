package nodes

import (
	"context"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"runtime"
	"testing"
	"time"
)

func TestHTTPClientPool_Client(t *testing.T) {
	pool := NewHTTPClientPool()

	{
		var origin = &serverconfigs.OriginConfig{
			Id:      1,
			Version: 2,
			Addr:    &serverconfigs.NetworkAddressConfig{Host: "127.0.0.1", PortRange: "1234"},
		}
		err := origin.Init(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		{
			client, err := pool.Client(nil, origin, origin.Addr.PickAddress(), nil, false)
			if err != nil {
				t.Fatal(err)
			}
			t.Log("client:", client)
		}
		for i := 0; i < 10; i++ {
			client, err := pool.Client(nil, origin, origin.Addr.PickAddress(), nil, false)
			if err != nil {
				t.Fatal(err)
			}
			t.Log("client:", client)
		}
	}
}

func TestHTTPClientPool_cleanClients(t *testing.T) {
	var origin = &serverconfigs.OriginConfig{
		Id:      1,
		Version: 2,
		Addr:    &serverconfigs.NetworkAddressConfig{Host: "127.0.0.1", PortRange: "1234"},
	}
	err := origin.Init(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	var pool = NewHTTPClientPool()

	for i := 0; i < 10; i++ {
		t.Log("get", i)
		_, _ = pool.Client(nil, origin, origin.Addr.PickAddress(), nil, false)
		time.Sleep(1 * time.Second)
	}
}

func BenchmarkHTTPClientPool_Client(b *testing.B) {
	runtime.GOMAXPROCS(1)

	var origin = &serverconfigs.OriginConfig{
		Id:      1,
		Version: 2,
		Addr:    &serverconfigs.NetworkAddressConfig{Host: "127.0.0.1", PortRange: "1234"},
	}
	err := origin.Init(context.Background())
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	var pool = NewHTTPClientPool()
	for i := 0; i < b.N; i++ {
		_, _ = pool.Client(nil, origin, origin.Addr.PickAddress(), nil, false)
	}
}

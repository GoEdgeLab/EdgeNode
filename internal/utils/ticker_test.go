package utils

import (
	"sync"
	"testing"
	"time"
)

func TestRawTicker(t *testing.T) {
	var ticker = time.NewTicker(2 * time.Second)
	go func() {
		for range ticker.C {
			t.Log("tick")
		}
		t.Log("stop")
	}()

	time.Sleep(6 * time.Second)
	ticker.Stop()
	time.Sleep(1 * time.Second)
}

func TestTicker(t *testing.T) {
	ticker := NewTicker(3 * time.Second)
	go func() {
		time.Sleep(10 * time.Second)
		ticker.Stop()
	}()
	for ticker.Next() {
		t.Log("tick")
	}
	t.Log("finished")
}

func TestTicker2(t *testing.T) {
	ticker := NewTicker(1 * time.Second)
	go func() {
		time.Sleep(5 * time.Second)
		ticker.Stop()
	}()
	for {
		t.Log("loop")
		select {
		case <-ticker.raw.C:
			t.Log("tick")
		case <-ticker.done:
			return
		}
	}
}

func TestTickerEvery(t *testing.T) {
	i := 0
	wg := &sync.WaitGroup{}
	wg.Add(1)
	Every(2*time.Second, func(ticker *Ticker) {
		i++
		t.Log("TestTickerEvery i:", i)
		if i >= 4 {
			ticker.Stop()
			wg.Done()
		}
	})
	wg.Wait()
}


func TestTicker_StopTwice(t *testing.T) {
	ticker := NewTicker(3 * time.Second)
	go func() {
		time.Sleep(10 * time.Second)
		ticker.Stop()
		ticker.Stop()
	}()
	for ticker.Next() {
		t.Log("tick")
	}
	t.Log("finished")
}

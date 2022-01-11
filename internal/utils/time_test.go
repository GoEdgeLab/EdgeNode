package utils

import (
	timeutil "github.com/iwind/TeaGo/utils/time"
	"testing"
	"time"
)

func TestUnixTime(t *testing.T) {
	for i := 0; i < 5; i++ {
		t.Log(UnixTime(), "real:", time.Now().Unix())
		time.Sleep(1 * time.Second)
	}
}

func TestGMTUnixTime(t *testing.T) {
	t.Log(GMTUnixTime(time.Now().Unix()))
}

func TestGMTTime(t *testing.T) {
	t.Log(GMTTime(time.Now()))
}

func TestFloorUnixTime(t *testing.T) {
	var timestamp = time.Now().Unix()
	t.Log("floor 60:", timestamp, FloorUnixTime(60), timeutil.FormatTime("Y-m-d H:i:s", FloorUnixTime(60)))
	t.Log("ceil 60:", timestamp, CeilUnixTime(60), timeutil.FormatTime("Y-m-d H:i:s", CeilUnixTime(60)))
	t.Log("floor 300:", timestamp, FloorUnixTime(300), timeutil.FormatTime("Y-m-d H:i:s", FloorUnixTime(300)))
	t.Log("next minute:", NextMinuteUnixTime())
}

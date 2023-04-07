package utils_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	timeutil "github.com/iwind/TeaGo/utils/time"
	"testing"
	"time"
)

func TestUnixTime(t *testing.T) {
	for i := 0; i < 5; i++ {
		t.Log(utils.UnixTime(), "real:", time.Now().Unix())
		time.Sleep(1 * time.Second)
	}
}

func TestGMTUnixTime(t *testing.T) {
	t.Log(utils.GMTUnixTime(time.Now().Unix()))
}

func TestGMTTime(t *testing.T) {
	t.Log(utils.GMTTime(time.Now()))
}

func TestFloorUnixTime(t *testing.T) {
	var timestamp = time.Now().Unix()
	t.Log("floor 60:", timestamp, utils.FloorUnixTime(60), timeutil.FormatTime("Y-m-d H:i:s", utils.FloorUnixTime(60)))
	t.Log("ceil 60:", timestamp, utils.CeilUnixTime(60), timeutil.FormatTime("Y-m-d H:i:s", utils.CeilUnixTime(60)))
	t.Log("floor 300:", timestamp, utils.FloorUnixTime(300), timeutil.FormatTime("Y-m-d H:i:s", utils.FloorUnixTime(300)))
	t.Log("next minute:", utils.NextMinuteUnixTime())
}

func TestYmd(t *testing.T) {
	t.Log(utils.Ymd())
}

func TestRound5Hi(t *testing.T) {
	t.Log(utils.Round5Hi())
}

package utils_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"testing"
	"time"
)

func TestGMTUnixTime(t *testing.T) {
	t.Log(utils.GMTUnixTime(time.Now().Unix()))
}

func TestGMTTime(t *testing.T) {
	t.Log(utils.GMTTime(time.Now()))
}

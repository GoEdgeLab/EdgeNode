package configs_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/configs"
	_ "github.com/iwind/TeaGo/bootstrap"
	"testing"
)

func TestLoadAPIConfig(t *testing.T) {
	config, err := configs.LoadAPIConfig()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", config)
}

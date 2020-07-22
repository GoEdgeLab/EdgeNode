package configs

import "testing"

func TestLoadAPIConfig(t *testing.T) {
	config, err := LoadAPIConfig()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(config)
}

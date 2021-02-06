package iplibrary

import "testing"

func TestIPv4RangeToCIDRRange(t *testing.T) {
	t.Log(iPv4RangeToCIDRRange("192.168.0.0", "192.168.255.255"))
}

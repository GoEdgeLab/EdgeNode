package utils

import "testing"

func TestIP2Long(t *testing.T) {
	t.Log(IP2Long("0.0.0.0"))
	t.Log(IP2Long("1.0.0.0"))
	t.Log(IP2Long("0.0.0.0.0"))
}

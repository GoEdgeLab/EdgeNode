package utils

import "testing"

func TestIP2Long(t *testing.T) {
	t.Log(IP2Long("0.0.0.0"))
	t.Log(IP2Long("1.0.0.0"))
	t.Log(IP2Long("0.0.0.0.0"))
	t.Log(IP2Long("2001:db8:0:1::101"))
	t.Log(IP2Long("2001:db8:0:1::102"))
	t.Log(IP2Long("::1"))
}

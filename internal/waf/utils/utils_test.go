package utils

import (
	"net/http"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestMatchStringCache(t *testing.T) {
	regex := regexp.MustCompile(`\d+`)
	t.Log(MatchStringCache(regex, "123"))
	t.Log(MatchStringCache(regex, "123"))
	t.Log(MatchStringCache(regex, "123"))
}

func TestMatchBytesCache(t *testing.T) {
	regex := regexp.MustCompile(`\d+`)
	t.Log(MatchBytesCache(regex, []byte("123")))
	t.Log(MatchBytesCache(regex, []byte("123")))
	t.Log(MatchBytesCache(regex, []byte("123")))
}

func TestMatchRemoteCache(t *testing.T) {
	client := http.Client{}
	for i := 0; i < 200_0000; i++ {
		req, err := http.NewRequest(http.MethodGet, "http://192.168.2.30:8882/?arg="+strconv.Itoa(i), nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("User-Agent", "GoTest/"+strconv.Itoa(i))
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
	}
}

func TestMatchBytesCache_WithoutCache(t *testing.T) {
	data := []byte(strings.Repeat("HELLO", 512))
	regex := regexp.MustCompile(`(?iU)\b(eval|system|exec|execute|passthru|shell_exec|phpinfo)\b`)
	before := time.Now()
	t.Log(regex.Match(data))
	t.Log(time.Since(before).Seconds()*1000, "ms")
}

func BenchmarkMatchStringCache(b *testing.B) {
	runtime.GOMAXPROCS(1)

	data := strings.Repeat("HELLO", 512)
	regex := regexp.MustCompile(`(?iU)\b(eval|system|exec|execute|passthru|shell_exec|phpinfo)\b`)
	_ = MatchStringCache(regex, data)

	for i := 0; i < b.N; i++ {
		_ = MatchStringCache(regex, data)
	}
}

func BenchmarkMatchStringCache_WithoutCache(b *testing.B) {
	runtime.GOMAXPROCS(1)

	data := strings.Repeat("HELLO", 512)
	regex := regexp.MustCompile(`(?iU)\b(eval|system|exec|execute|passthru|shell_exec|phpinfo)\b`)

	for i := 0; i < b.N; i++ {
		_ = regex.MatchString(data)
	}
}

func BenchmarkMatchBytesCache_WithoutCache(b *testing.B) {
	runtime.GOMAXPROCS(1)

	data := []byte(strings.Repeat("HELLO", 512))
	regex := regexp.MustCompile(`(?iU)\b(eval|system|exec|execute|passthru|shell_exec|phpinfo)\b`)

	for i := 0; i < b.N; i++ {
		_ = regex.Match(data)
	}
}

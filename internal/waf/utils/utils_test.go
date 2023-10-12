package utils_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/re"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	"github.com/TeaOSLab/EdgeNode/internal/waf/utils"
	"github.com/iwind/TeaGo/rands"
	"net/http"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestMatchStringCache(t *testing.T) {
	regex := re.MustCompile(`\d+`)
	t.Log(utils.MatchStringCache(regex, "123", utils.CacheShortLife))
	t.Log(utils.MatchStringCache(regex, "123", utils.CacheShortLife))
	t.Log(utils.MatchStringCache(regex, "123", utils.CacheShortLife))
}

func TestMatchBytesCache(t *testing.T) {
	regex := re.MustCompile(`\d+`)
	t.Log(utils.MatchBytesCache(regex, []byte("123"), utils.CacheShortLife))
	t.Log(utils.MatchBytesCache(regex, []byte("123"), utils.CacheShortLife))
	t.Log(utils.MatchBytesCache(regex, []byte("123"), utils.CacheShortLife))
}

func TestMatchRemoteCache(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var client = http.Client{}
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

	var data = strings.Repeat("HELLO", 128)
	var regex = re.MustCompile(`(?iU)\b(eval|system|exec|execute|passthru|shell_exec|phpinfo)\b`)
	//b.Log(regex.Keywords())
	_ = utils.MatchStringCache(regex, data, utils.CacheShortLife)

	for i := 0; i < b.N; i++ {
		_ = utils.MatchStringCache(regex, data, utils.CacheShortLife)
	}
}

func BenchmarkMatchStringCache_LowHit(b *testing.B) {
	runtime.GOMAXPROCS(1)

	var regex = re.MustCompile(`(?iU)\b(eval|system|exec|execute|passthru|shell_exec|phpinfo)\b`)
	//b.Log(regex.Keywords())

	for i := 0; i < b.N; i++ {
		var data = strings.Repeat("A", rands.Int(0, 100))
		_ = utils.MatchStringCache(regex, data, utils.CacheShortLife)
	}
}

func BenchmarkMatchStringCache_WithoutCache(b *testing.B) {
	runtime.GOMAXPROCS(1)

	data := strings.Repeat("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.110 Safari/537.36", 8)
	regex := re.MustCompile(`(?iU)\b(eval|system|exec|execute|passthru|shell_exec|phpinfo)\b`)
	for i := 0; i < b.N; i++ {
		_ = regex.MatchString(data)
	}
}

func BenchmarkMatchStringCache_WithoutCache2(b *testing.B) {
	runtime.GOMAXPROCS(1)

	data := strings.Repeat("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.110 Safari/537.36", 8)
	regex := regexp.MustCompile(`(?iU)\b(eval|system|exec|execute|passthru|shell_exec|phpinfo)\b`)
	for i := 0; i < b.N; i++ {
		_ = regex.MatchString(data)
	}
}

func BenchmarkMatchBytesCache_WithoutCache(b *testing.B) {
	runtime.GOMAXPROCS(1)

	data := []byte(strings.Repeat("HELLO", 128))
	regex := re.MustCompile(`(?iU)\b(eval|system|exec|execute|passthru|shell_exec|phpinfo)\b`)

	for i := 0; i < b.N; i++ {
		_ = regex.Match(data)
	}
}

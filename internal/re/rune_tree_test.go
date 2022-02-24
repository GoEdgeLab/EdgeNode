// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package re_test

import (
	"github.com/TeaOSLab/EdgeNode/internal/re"
	"github.com/iwind/TeaGo/assert"
	"regexp"
	"testing"
)

func TestNewRuneTree(t *testing.T) {
	var a = assert.NewAssertion(t)

	var tree = re.NewRuneTree([]string{"abc", "abd", "def", "GHI", "中国", "@"})
	a.IsTrue(tree.Lookup("ABC", true))
	a.IsTrue(tree.Lookup("ABC1", true))
	a.IsTrue(tree.Lookup("1ABC", true))
	a.IsTrue(tree.Lookup("def", true))
	a.IsTrue(tree.Lookup("ghI", true))
	a.IsFalse(tree.Lookup("d ef", true))
	a.IsFalse(tree.Lookup("de", true))
	a.IsFalse(tree.Lookup("de f", true))
	a.IsTrue(tree.Lookup("我是中国人", true))
	a.IsTrue(tree.Lookup("iwind.liu@gmail.com", true))
}

func TestNewRuneTree2(t *testing.T) {
	var tree = re.NewRuneTree([]string{"abc", "abd", "def", "GHI", "中国", "@"})
	tree.Lookup("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.110 Safari/537.36", true)
}

func BenchmarkRuneMap_Lookup(b *testing.B) {
	var tree = re.NewRuneTree([]string{"abc", "abd", "def", "ghi", "中国"})
	for i := 0; i < b.N; i++ {
		tree.Lookup("我来自中国", true)
	}
}

func BenchmarkRuneMap_Lookup2_NOT_FOUND(b *testing.B) {
	var tree = re.NewRuneTree([]string{"abc", "abd", "cde", "GHI"})
	for i := 0; i < b.N; i++ {
		tree.Lookup("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.110 Safari/537.36", true)
	}
}

func BenchmarkRune_Regexp_FOUND(b *testing.B) {
	var reg = regexp.MustCompile("(?i)abc|abd|cde|GHI")
	for i := 0; i < b.N; i++ {
		reg.MatchString("HELLO WORLD ABC 123 456 abc HELLO WORLD HELLO WORLD ABC 123 456 abc HELLO WORLD HELLO WORLD ABC 123 456 abc HELLO WORLD")
	}
}

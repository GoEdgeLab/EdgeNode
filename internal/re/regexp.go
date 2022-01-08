// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package re

import (
	"regexp"
	"strings"
)

var prefixReg = regexp.MustCompile(`^\(\?([\w\s]+)\)`) // (?x)
var prefixReg2 = regexp.MustCompile(`^\(\?([\w\s]*:)`) // (?x: ...
var braceZero = regexp.MustCompile(`^{\s*0*\s*}`)      // {0}
var braceZero2 = regexp.MustCompile(`^{\s*0*\s*,`)     // {0, x}

type Regexp struct {
	exp       string
	rawRegexp *regexp.Regexp

	isStrict          bool
	isCaseInsensitive bool
	keywords          []string
	keywordsMap       RuneMap
}

func MustCompile(exp string) *Regexp {
	var reg = &Regexp{
		exp:       exp,
		rawRegexp: regexp.MustCompile(exp),
	}
	reg.init()
	return reg
}

func Compile(exp string) (*Regexp, error) {
	reg, err := regexp.Compile(exp)
	if err != nil {
		return nil, err
	}
	return NewRegexp(reg), nil
}

func NewRegexp(rawRegexp *regexp.Regexp) *Regexp {
	var reg = &Regexp{
		exp:       rawRegexp.String(),
		rawRegexp: rawRegexp,
	}
	reg.init()
	return reg
}

func (this *Regexp) init() {
	if len(this.exp) == 0 {
		return
	}

	//var keywords = []string{}

	var exp = strings.TrimSpace(this.exp)

	// 去掉前面的(?...)
	if prefixReg.MatchString(exp) {
		var matches = prefixReg.FindStringSubmatch(exp)
		var modifiers = matches[1]
		if strings.Contains(modifiers, "i") {
			this.isCaseInsensitive = true
		}
		exp = exp[len(matches[0]):]
	}

	var keywords = this.ParseKeywords(exp)
	this.keywords = keywords
	if len(keywords) > 0 {
		this.keywordsMap = NewRuneTree(keywords)
	}
}

func (this *Regexp) Keywords() []string {
	return this.keywords
}

func (this *Regexp) Raw() *regexp.Regexp {
	return this.rawRegexp
}

func (this *Regexp) IsCaseInsensitive() bool {
	return this.isCaseInsensitive
}

func (this *Regexp) MatchString(s string) bool {
	if this.keywordsMap != nil {
		var b = this.keywordsMap.Lookup(s, this.isCaseInsensitive)
		if !b {
			return false
		}
		if this.isStrict {
			return true
		}
	}
	return this.rawRegexp.MatchString(s)
}

func (this *Regexp) Match(s []byte) bool {
	if this.keywordsMap != nil {
		var b = this.keywordsMap.Lookup(string(s), this.isCaseInsensitive)
		if !b {
			return false
		}
		if this.isStrict {
			return true
		}
	}
	return this.rawRegexp.Match(s)
}

// ParseKeywords 提取表达式中的关键词
// TODO 支持嵌套，类似于 A(abc|bcd)
// TODO 支持 (?:xxx)
// TODO 支持  （abc)(bcd)(efg)
func (this *Regexp) ParseKeywords(exp string) []string {
	var keywords = []string{}
	if len(exp) == 0 {
		return nil
	}

	var runes = []rune(exp)

	// (a|b|c)
	reg, err := regexp.Compile(exp)
	if err == nil {
		var countSub = reg.NumSubexp()
		if countSub == 1 {
			beginIndex := this.indexOfSymbol(runes, '(')
			if beginIndex >= 0 {
				runes = runes[beginIndex+1:]
				symbolIndex := this.indexOfSymbol(runes, ')')
				if symbolIndex > 0 && this.isPlain(runes[symbolIndex+1:]) {
					runes = runes[:symbolIndex]
					if len(runes) == 0 {
						return nil
					}
				}
			}
		}
	}

	var lastIndex = 0
	for index, r := range runes {
		if r == '|' {
			if index > 0 && runes[index-1] != '\\' {
				var ks = this.parseKeyword(runes[lastIndex:index])
				if len(ks) > 0 {
					keywords = append(keywords, string(ks))
				} else {
					return nil
				}
				lastIndex = index + 1
			}
		}
	}
	if lastIndex == 0 {
		var ks = this.parseKeyword(runes)
		if len(ks) > 0 {
			keywords = append(keywords, string(ks))
		} else {
			return nil
		}
	} else if lastIndex > 0 {
		var ks = this.parseKeyword(runes[lastIndex:])
		if len(ks) > 0 {
			keywords = append(keywords, string(ks))
		} else {
			return nil
		}
	}
	return keywords
}

func (this *Regexp) parseKeyword(keyword []rune) (result []rune) {
	if len(keyword) == 0 {
		return
	}

	// remove first \b
	for index, r := range keyword {
		if r == '\b' {
			keyword = keyword[index+1:]
			break
		} else if r != '\t' && r != '\r' && r != '\n' && r != ' ' {
			break
		}
	}
	if len(keyword) == 0 {
		return
	}

	for index, r := range keyword {
		if index == 0 && r == '^' {
			continue
		}
		if r == '(' || r == ')' {
			if index == 0 {
				return nil
			}
			if keyword[index-1] != '\\' {
				return nil
			}
		}
		if r == '[' || r == '{' || r == '.' || r == '+' || r == '$' {
			if index == 0 {
				return nil
			}
			if keyword[index-1] != '\\' {
				if r == '{' && (braceZero.MatchString(string(keyword[index:])) || braceZero2.MatchString(string(keyword[index:]))) { // r {0, ...}
					return result[:len(result)-1]
				}

				return
			}
		}
		if r == '?' || r == '*' {
			if index == 0 {
				return nil
			}
			return result[:len(result)-1]
		}
		if r == '\\' || r == '\b' {
			// TODO 将来更精细的处理 \d, \s, \$等
			break
		}

		result = append(result, r)
	}
	return
}

// 查找符号位置
func (this *Regexp) indexOfSymbol(runes []rune, symbol rune) int {
	for index, c := range runes {
		if c == symbol && (index == 0 || runes[index-1] != '\\') {
			return index
		}
	}
	return -1
}

// 是否可视为为普通字符
func (this *Regexp) isPlain(runes []rune) bool {
	for _, r := range []rune{'|', '(', ')'} {
		if this.indexOfSymbol(runes, r) >= 0 {
			return false
		}
	}
	return true
}

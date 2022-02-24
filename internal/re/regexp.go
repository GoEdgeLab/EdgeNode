// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package re

import (
	"regexp"
	"regexp/syntax"
	"strings"
)

var prefixReg = regexp.MustCompile(`^\(\?([\w\s]+)\)`) // (?x)
var prefixReg2 = regexp.MustCompile(`^\(\?([\w\s]*:)`) // (?x: ...
var braceZeroReg = regexp.MustCompile(`^{\s*0*\s*}`)   // {0}
var braceZeroReg2 = regexp.MustCompile(`^{\s*0*\s*,`)  // {0, x}

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

	var filteredKeywords = []string{}
	var minLength = 1
	var isValid = true
	for _, keyword := range keywords {
		if len(keyword) <= minLength {
			isValid = false
			break
		}
	}
	if isValid {
		filteredKeywords = keywords
	}

	this.keywords = filteredKeywords
	if len(filteredKeywords) > 0 {
		this.keywordsMap = NewRuneTree(filteredKeywords)
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
func (this *Regexp) ParseKeywords(exp string) (keywords []string) {
	if len(exp) == 0 {
		return nil
	}

	reg, err := syntax.Parse(exp, syntax.Perl)
	if err != nil {
		return nil
	}

	if len(reg.Sub) == 0 {
		var keywordRunes = this.parseKeyword(reg.String())
		if len(keywordRunes) > 0 {
			keywords = append(keywords, string(keywordRunes))
		}
		return
	}
	if len(reg.Sub) == 1 {
		if reg.Op == syntax.OpStar || reg.Op == syntax.OpQuest || reg.Op == syntax.OpRepeat {
			return nil
		}
		return this.ParseKeywords(reg.Sub[0].String())
	}

	switch reg.Op {
	case syntax.OpConcat:
		var prevKeywords = []string{}
		var isStarted bool
		for _, sub := range reg.Sub {
			if sub.String() == `\b` {
				if isStarted {
					break
				}
				continue
			}
			if sub.Op != syntax.OpLiteral && sub.Op != syntax.OpCapture && sub.Op != syntax.OpAlternate {
				if isStarted {
					break
				}
				continue
			}
			var subKeywords = this.ParseKeywords(sub.String())
			if len(subKeywords) > 0 {
				if !isStarted {
					prevKeywords = subKeywords
					isStarted = true
				} else {
					for _, prevKeyword := range prevKeywords {
						for _, subKeyword := range subKeywords {
							keywords = append(keywords, prevKeyword+subKeyword)
						}
					}
					prevKeywords = keywords
				}
			} else {
				break
			}
		}
		if len(prevKeywords) > 0 && len(keywords) == 0 {
			keywords = prevKeywords
		}
	case syntax.OpAlternate:
		for _, sub := range reg.Sub {
			var subKeywords = this.ParseKeywords(sub.String())
			if len(subKeywords) == 0 {
				keywords = nil
				return
			}
			keywords = append(keywords, subKeywords...)
		}
	}

	return
}

func (this *Regexp) parseKeyword(subExp string) (result []rune) {
	if len(subExp) == 0 {
		return nil
	}

	// 去除开始和结尾的()
	if subExp[0] == '(' && subExp[len(subExp)-1] == ')' {
		subExp = subExp[1 : len(subExp)-1]
		if len(subExp) == 0 {
			return
		}
	}

	var runes = []rune(subExp)

	for index, r := range runes {
		if r == '[' || r == '{' || r == '.' || r == '+' || r == '$' {
			if index == 0 {
				return
			}
			if runes[index-1] != '\\' {
				if r == '{' && (braceZeroReg.MatchString(subExp[index:])) || braceZeroReg2.MatchString(subExp[index:]) { // r {0, ...}
					if len(result) == 0 {
						return nil
					}
					return result[:len(result)-1]
				}

				return
			}
		}
		if r == '?' || r == '*' {
			if index == 0 {
				return
			}
			if runes[index-1] != '\\' {
				if len(result) > 0 {
					return result[:len(result)-1]
				}
				return
			}
		}

		if (r == 'n' || r == 't' || r == 'a' || r == 'f' || r == 'r' || r == 'v' || r == 'x') && index > 0 && runes[index-1] == '\\' {
			switch r {
			case 'n':
				r = '\n'
			case 't':
				r = '\t'
			case 'f':
				r = '\f'
			case 'r':
				r = '\r'
			case 'v':
				r = '\v'
			case 'a':
				r = '\a'
			case 'x':
				return
			}
		}

		if r == '\\' {
			continue
		}
		result = append(result, r)
	}

	return
}

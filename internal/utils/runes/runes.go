// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package runes

// ContainsAnyWordRunes 直接使用rune检查字符串是否包含任一单词
func ContainsAnyWordRunes(s string, words [][]rune, isCaseInsensitive bool) bool {
	var allRunes = []rune(s)
	if len(allRunes) == 0 || len(words) == 0 {
		return false
	}

	var lastRune rune  // last searching rune in s
	var lastIndex = -2 // -2: not started, -1: not found, >=0: rune index
	for _, wordRunes := range words {
		if len(wordRunes) == 0 {
			continue
		}

		if lastIndex > -2 && lastRune == wordRunes[0] {
			if lastIndex >= 0 {
				result, _ := ContainsWordRunes(allRunes[lastIndex:], wordRunes, isCaseInsensitive)
				if result {
					return true
				}
			}
			continue
		} else {
			result, firstIndex := ContainsWordRunes(allRunes, wordRunes, isCaseInsensitive)
			lastIndex = firstIndex
			if result {
				return true
			}
		}
		lastRune = wordRunes[0]
	}
	return false
}

// ContainsAnyWord 检查字符串是否包含任一单词
func ContainsAnyWord(s string, words []string, isCaseInsensitive bool) bool {
	var allRunes = []rune(s)
	if len(allRunes) == 0 || len(words) == 0 {
		return false
	}

	var lastRune rune  // last searching rune in s
	var lastIndex = -2 // -2: not started, -1: not found, >=0: rune index
	for _, word := range words {
		var wordRunes = []rune(word)
		if len(wordRunes) == 0 {
			continue
		}

		if lastIndex > -2 && lastRune == wordRunes[0] {
			if lastIndex >= 0 {
				result, _ := ContainsWordRunes(allRunes[lastIndex:], wordRunes, isCaseInsensitive)
				if result {
					return true
				}
			}
			continue
		} else {
			result, firstIndex := ContainsWordRunes(allRunes, wordRunes, isCaseInsensitive)
			lastIndex = firstIndex
			if result {
				return true
			}
		}
		lastRune = wordRunes[0]
	}
	return false
}

// ContainsAllWords 检查字符串是否包含所有单词
func ContainsAllWords(s string, words []string, isCaseInsensitive bool) bool {
	var allRunes = []rune(s)
	if len(allRunes) == 0 || len(words) == 0 {
		return false
	}

	for _, word := range words {
		if result, _ := ContainsWordRunes(allRunes, []rune(word), isCaseInsensitive); !result {
			return false
		}
	}
	return true
}

// ContainsWordRunes 检查字符列表是否包含某个单词子字符列表
func ContainsWordRunes(allRunes []rune, subRunes []rune, isCaseInsensitive bool) (result bool, firstIndex int) {
	firstIndex = -1

	var l = len(subRunes)
	if l == 0 {
		return false, 0
	}

	var al = len(allRunes)

	for index, r := range allRunes {
		if EqualRune(r, subRunes[0], isCaseInsensitive) && (index == 0 || !isChar(allRunes[index-1]) /**boundary check **/) {
			if firstIndex < 0 {
				firstIndex = index
			}

			var found = true
			if l > 1 {
				for i := 1; i < l; i++ {
					var subIndex = index + i
					if subIndex > al-1 || !EqualRune(allRunes[subIndex], subRunes[i], isCaseInsensitive) {
						found = false
						break
					}
				}
			}

			// check after charset
			if found && (al <= index+l || !isChar(allRunes[index+l]) /**boundary check **/) {
				return true, firstIndex
			}
		}
	}

	return false, firstIndex
}

// ContainsSubRunes 检查字符列表是否包含某个子子字符列表
// 与 ContainsWordRunes 不同，这里不需要检查边界符号
func ContainsSubRunes(allRunes []rune, subRunes []rune, isCaseInsensitive bool) bool {
	var l = len(subRunes)
	if l == 0 {
		return false
	}

	var al = len(allRunes)

	for index, r := range allRunes {
		if EqualRune(r, subRunes[0], isCaseInsensitive) {
			var found = true
			if l > 1 {
				for i := 1; i < l; i++ {
					var subIndex = index + i
					if subIndex > al-1 || !EqualRune(allRunes[subIndex], subRunes[i], isCaseInsensitive) {
						found = false
						break
					}
				}
			}

			// check after charset
			if found {
				return true
			}
		}
	}

	return false
}

// EqualRune 判断两个rune是否相同
func EqualRune(r1 rune, r2 rune, isCaseInsensitive bool) bool {
	const d = 'a' - 'A'
	return r1 == r2 ||
		(isCaseInsensitive && r1 >= 'a' && r1 <= 'z' && r1-r2 == d) ||
		(isCaseInsensitive && r1 >= 'A' && r1 <= 'Z' && r1-r2 == -d)
}

func isChar(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
}

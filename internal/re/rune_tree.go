// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package re

type RuneMap map[rune]*RuneTree

func (this *RuneMap) Lookup(s string, caseInsensitive bool) bool {
	return this.lookup([]rune(s), caseInsensitive, 0)
}

func (this RuneMap) lookup(runes []rune, caseInsensitive bool, depth int) bool {
	if len(runes) == 0 {
		return false
	}
	for i, r := range runes {
		tree, ok := this[r]
		if !ok {
			if caseInsensitive {
				if r >= 'a' && r <= 'z' {
					r -= 32
					tree, ok = this[r]
				} else if r >= 'A' && r <= 'Z' {
					r += 32
					tree, ok = this[r]
				}
			}
			if !ok {
				if depth > 0 {
					return false
				}
				continue
			}
		}
		if tree.IsEnd {
			return true
		}
		b := tree.Children.lookup(runes[i+1:], caseInsensitive, depth+1)
		if b {
			return true
		}
	}
	return false
}

type RuneTree struct {
	Children RuneMap
	IsEnd    bool
}

func NewRuneTree(list []string) RuneMap {
	var rootMap = RuneMap{}
	for _, s := range list {
		if len(s) == 0 {
			continue
		}

		var lastMap = rootMap
		var runes = []rune(s)
		for index, r := range runes {
			tree, ok := lastMap[r]
			if !ok {
				tree = &RuneTree{
					Children: RuneMap{},
				}
				lastMap[r] = tree
			}
			if index == len(runes)-1 {
				tree.IsEnd = true
			}
			lastMap = tree.Children
		}
	}
	return rootMap
}

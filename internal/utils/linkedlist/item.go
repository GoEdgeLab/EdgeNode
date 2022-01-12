// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package linkedlist

type Item struct {
	prev *Item
	next *Item

	Value interface{}
}

func NewItem(value interface{}) *Item {
	return &Item{Value: value}
}

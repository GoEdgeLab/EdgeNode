// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package linkedlist

type List struct {
	head  *Item
	end   *Item
	count int
}

func NewList() *List {
	return &List{}
}

func (this *List) Head() *Item {
	return this.head
}

func (this *List) End() *Item {
	return this.end
}

func (this *List) Push(item *Item) {
	if item == nil {
		return
	}

	// 如果已经在末尾了，则do nothing
	if this.end == item {
		return
	}

	if item.prev != nil || item.next != nil || this.head == item {
		this.Remove(item)
	}
	this.add(item)
}

func (this *List) Remove(item *Item) {
	if item == nil {
		return
	}
	if item.prev != nil {
		item.prev.next = item.next
	}
	if item.next != nil {
		item.next.prev = item.prev
	}
	if item == this.head {
		this.head = item.next
	}
	if item == this.end {
		this.end = item.prev
	}

	item.prev = nil
	item.next = nil
	this.count--
}

func (this *List) Len() int {
	return this.count
}

func (this *List) Range(f func(item *Item) (goNext bool)) {
	for e := this.head; e != nil; e = e.next {
		goNext := f(e)
		if !goNext {
			break
		}
	}
}

func (this *List) Reset() {
	this.head = nil
	this.end = nil
}

func (this *List) add(item *Item) {
	if item == nil {
		return
	}
	if this.end != nil {
		this.end.next = item
		item.prev = this.end
		item.next = nil
	}
	this.end = item
	if this.head == nil {
		this.head = item
	}
	this.count++
}

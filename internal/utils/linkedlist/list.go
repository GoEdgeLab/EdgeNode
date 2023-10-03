// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package linkedlist

type List[T any]  struct {
	head  *Item[T]
	end   *Item[T]
	count int
}

func NewList[T any]() *List[T] {
	return &List[T]{}
}

func (this *List[T]) Head() *Item[T] {
	return this.head
}

func (this *List[T]) End() *Item[T] {
	return this.end
}

func (this *List[T]) Push(item *Item[T]) {
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

func (this *List[T]) Remove(item *Item[T]) {
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

func (this *List[T]) Len() int {
	return this.count
}

func (this *List[T]) Range(f func(item *Item[T]) (goNext bool)) {
	for e := this.head; e != nil; e = e.next {
		goNext := f(e)
		if !goNext {
			break
		}
	}
}

func (this *List[T]) Reset() {
	this.head = nil
	this.end = nil
}

func (this *List[T]) add(item *Item[T]) {
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

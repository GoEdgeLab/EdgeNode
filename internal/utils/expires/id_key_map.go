// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package expires

type IdKeyMap struct {
	idKeys map[int64]string // id => key
	keyIds map[string]int64 // key => id
}

func NewIdKeyMap() *IdKeyMap {
	return &IdKeyMap{
		idKeys: map[int64]string{},
		keyIds: map[string]int64{},
	}
}

func (this *IdKeyMap) Add(id int64, key string) {
	oldKey, ok := this.idKeys[id]
	if ok {
		delete(this.keyIds, oldKey)
	}

	oldId, ok := this.keyIds[key]
	if ok {
		delete(this.idKeys, oldId)
	}

	this.idKeys[id] = key
	this.keyIds[key] = id
}

func (this *IdKeyMap) Key(id int64) (key string, ok bool) {
	key, ok = this.idKeys[id]
	return
}

func (this *IdKeyMap) Id(key string) (id int64, ok bool) {
	id, ok = this.keyIds[key]
	return
}

func (this *IdKeyMap) DeleteId(id int64) {
	key, ok := this.idKeys[id]
	if ok {
		delete(this.keyIds, key)
	}
	delete(this.idKeys, id)
}

func (this *IdKeyMap) DeleteKey(key string) {
	id, ok := this.keyIds[key]
	if ok {
		delete(this.idKeys, id)
	}
	delete(this.keyIds, key)
}

func (this *IdKeyMap) Len() int {
	return len(this.idKeys)
}

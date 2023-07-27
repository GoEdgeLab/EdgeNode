package events

import (
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"sync"
)

type Callbacks = []func()

var eventsMap = map[Event]map[interface{}]Callbacks{} // event => map[event key][]callback
var locker = sync.Mutex{}

var eventKeyId = 0

func NewKey() interface{} {
	locker.Lock()
	defer locker.Unlock()
	eventKeyId++
	return eventKeyId
}

// On 增加事件回调
func On(event Event, callback func()) {
	OnKey(event, nil, callback)
}

func OnEvents(events []Event, callback func()) {
	for _, event := range events {
		On(event, callback)
	}
}

func OnClose(callback func()) {
	On(EventQuit, callback)
	On(EventTerminated, callback)
}

// OnKey 使用Key增加事件回调
func OnKey(event Event, key interface{}, callback func()) {
	if key == nil {
		key = NewKey()
	}

	locker.Lock()
	defer locker.Unlock()

	m, ok := eventsMap[event]
	if !ok {
		m = map[interface{}]Callbacks{}
		eventsMap[event] = m
	}
	m[key] = append(m[key], callback)
}

// Remove 删除事件回调
func Remove(key interface{}) {
	if key == nil {
		return
	}

	locker.Lock()
	for k, m := range eventsMap {
		_, ok := m[key]
		if ok {
			delete(m, key)
			eventsMap[k] = m
		}
	}
	locker.Unlock()
}

// Notify 通知事件
func Notify(event Event) {
	// 特殊事件
	switch event {
	case EventQuit:
		teaconst.IsQuiting = true
	case EventTerminated:
		teaconst.IsQuiting = true
	}

	locker.Lock()
	m := eventsMap[event]
	locker.Unlock()

	for _, callbacks := range m {
		for _, callback := range callbacks {
			callback()
		}
	}
}

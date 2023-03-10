// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package monitor

import (
	"encoding/json"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/iwind/TeaGo/maps"
	"time"
)

var SharedValueQueue = NewValueQueue()

func init() {
	if !teaconst.IsMain {
		return
	}

	events.On(events.EventLoaded, func() {
		goman.New(func() {
			SharedValueQueue.Start()
		})
	})
}

// ValueQueue 数据记录队列
type ValueQueue struct {
	valuesChan chan *ItemValue
}

func NewValueQueue() *ValueQueue {
	return &ValueQueue{
		valuesChan: make(chan *ItemValue, 1024),
	}
}

// Start 启动队列
func (this *ValueQueue) Start() {
	// 这里单次循环就行，因为Loop里已经使用了Range通道
	err := this.Loop()
	if err != nil {
		remotelogs.ErrorObject("MONITOR_QUEUE", err)
	}
}

// Add 添加数据
func (this *ValueQueue) Add(item string, value maps.Map) {
	valueJSON, err := json.Marshal(value)
	if err != nil {
		remotelogs.Error("MONITOR_QUEUE", "marshal value error: "+err.Error())
		return
	}
	select {
	case this.valuesChan <- &ItemValue{
		Item:      item,
		ValueJSON: valueJSON,
		CreatedAt: time.Now().Unix(),
	}:
	default:

	}
}

// Loop 单次循环
func (this *ValueQueue) Loop() error {
	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return err
	}

	for value := range this.valuesChan {
		_, err = rpcClient.NodeValueRPC.CreateNodeValue(rpcClient.Context(), &pb.CreateNodeValueRequest{
			Item:      value.Item,
			ValueJSON: value.ValueJSON,
			CreatedAt: value.CreatedAt,
		})
		if err != nil {
			if rpc.IsConnError(err) {
				remotelogs.Warn("MONITOR", err.Error())
			} else {
				remotelogs.Error("MONITOR", err.Error())
			}
			continue
		}
	}
	return nil
}

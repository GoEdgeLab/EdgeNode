// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package metrics_test

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/metrics"
	"testing"
)

func TestNewManager(t *testing.T) {
	var manager = metrics.NewManager()
	{
		manager.Update([]*serverconfigs.MetricItemConfig{})
		for _, task := range manager.TaskMap() {
			t.Log(task.Item().Id)
		}
	}
	{
		t.Log("====")
		manager.Update([]*serverconfigs.MetricItemConfig{
			{
				Id: 1,
			},
			{
				Id: 2,
			},
			{
				Id: 3,
			},
		})
		for _, task := range manager.TaskMap() {
			t.Log("task:", task.Item().Id)
		}
	}

	{
		t.Log("====")
		manager.Update([]*serverconfigs.MetricItemConfig{
			{
				Id: 1,
			},
			{
				Id: 2,
			},
		})
		for _, task := range manager.TaskMap() {
			t.Log("task:", task.Item().Id)
		}
	}

	{
		t.Log("====")
		manager.Update([]*serverconfigs.MetricItemConfig{
			{
				Id:      1,
				Version: 1,
			},
		})
		for _, task := range manager.TaskMap() {
			t.Log("task:", task.Item().Id)
		}
	}
}

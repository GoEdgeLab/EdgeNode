// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package metrics_test

import (
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/metrics"
	_ "github.com/iwind/TeaGo/bootstrap"
	"github.com/iwind/TeaGo/rands"
	"testing"
	"time"
)

type testObj struct {
	ip string
}

func (this *testObj) MetricKey(key string) string {
	return this.ip
}

func (this *testObj) MetricValue(value string) (int64, bool) {
	return 1, true
}

func (this *testObj) MetricServerId() int64 {
	return int64(rands.Int(1, 100))
}

func (this *testObj) MetricCategory() string {
	return "http"
}

func TestTask_Init(t *testing.T) {
	var task = metrics.NewTask(&serverconfigs.MetricItemConfig{
		Id:         1,
		IsOn:       false,
		Category:   "",
		Period:     0,
		PeriodUnit: "",
		Keys:       nil,
		Value:      "",
	})
	err := task.Init()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = task.Stop()
	}()
	t.Log("ok")
}

func TestTask_Add(t *testing.T) {
	var task = metrics.NewTask(&serverconfigs.MetricItemConfig{
		Id:         1,
		IsOn:       false,
		Category:   "",
		Period:     1,
		PeriodUnit: serverconfigs.MetricItemPeriodUnitDay,
		Keys:       []string{"${remoteAddr}"},
		Value:      "${countRequest}",
	})
	err := task.Init()
	if err != nil {
		t.Fatal(err)
	}
	err = task.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = task.Stop()
	}()

	task.Add(&testObj{ip: "127.0.0.2"})
	time.Sleep(1 * time.Second) // waiting for inserting
}

func TestTask_Add_Many(t *testing.T) {
	var task = metrics.NewTask(&serverconfigs.MetricItemConfig{
		Id:         1,
		IsOn:       false,
		Category:   "",
		Period:     1,
		PeriodUnit: serverconfigs.MetricItemPeriodUnitDay,
		Keys:       []string{"${remoteAddr}"},
		Value:      "${countRequest}",
		Version:    1,
	})
	err := task.Init()
	if err != nil {
		t.Fatal(err)
	}
	err = task.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = task.Stop()
	}()

	for i := 0; i < 4_000_000; i++ {
		task.Add(&testObj{
			ip: fmt.Sprintf("%d.%d.%d.%d", rands.Int(0, 255), rands.Int(0, 255), rands.Int(0, 255), rands.Int(0, 255)),
		})
		if i%10000 == 0 {
			time.Sleep(1 * time.Second)
		}
	}
}

func TestTask_InsertStat(t *testing.T) {
	var item = &serverconfigs.MetricItemConfig{
		Id:         1,
		IsOn:       false,
		Category:   "",
		Period:     1,
		PeriodUnit: serverconfigs.MetricItemPeriodUnitDay,
		Keys:       []string{"${remoteAddr}"},
		Value:      "${countRequest}",
		Version:    1,
	}
	var task = metrics.NewTask(item)
	err := task.Init()
	if err != nil {
		t.Fatal(err)
	}
	err = task.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = task.Stop()
	}()

	err = task.InsertStat(&metrics.Stat{
		ServerId: 1,
		Keys:     []string{"127.0.0.1"},
		Hash:     "",
		Value:    1,
		Time:     item.CurrentTime(),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}

func TestTask_CleanExpired(t *testing.T) {
	var task = metrics.NewTask(&serverconfigs.MetricItemConfig{
		Id:         1,
		IsOn:       false,
		Category:   "",
		Period:     1,
		PeriodUnit: serverconfigs.MetricItemPeriodUnitDay,
		Keys:       []string{"${remoteAddr}"},
		Value:      "${countRequest}",
		Version:    1,
	})
	err := task.Init()
	if err != nil {
		t.Fatal(err)
	}
	err = task.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = task.Stop()
	}()

	err = task.CleanExpired()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}

func TestTask_Upload(t *testing.T) {
	var task = metrics.NewTask(&serverconfigs.MetricItemConfig{
		Id:         1,
		IsOn:       false,
		Category:   "",
		Period:     1,
		PeriodUnit: serverconfigs.MetricItemPeriodUnitDay,
		Keys:       []string{"${remoteAddr}"},
		Value:      "${countRequest}",
		Version:    1,
	})
	err := task.Init()
	if err != nil {
		t.Fatal(err)
	}
	err = task.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = task.Stop()
	}()

	err = task.Upload(0)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}

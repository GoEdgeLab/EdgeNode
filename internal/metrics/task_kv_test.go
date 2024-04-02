// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package metrics_test

import (
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/configutils"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/metrics"
	"github.com/TeaOSLab/EdgeNode/internal/utils/testutils"
	_ "github.com/iwind/TeaGo/bootstrap"
	"github.com/iwind/TeaGo/rands"
	"log"
	"runtime"
	"sync"
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

func TestKVTask_Init(t *testing.T) {
	var task = metrics.NewKVTask(&serverconfigs.MetricItemConfig{
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

func TestKVTask_Add(t *testing.T) {
	var task = metrics.NewKVTask(&serverconfigs.MetricItemConfig{
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

	if testutils.IsSingleTesting() {
		time.Sleep(1 * time.Second) // waiting for inserting
	}
}

func TestKVTask_Add_Many(t *testing.T) {
	if !testutils.IsSingleTesting() {
		return
	}

	var task = metrics.NewKVTask(&serverconfigs.MetricItemConfig{
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

func TestKVTask_InsertStat(t *testing.T) {
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
	var task = metrics.NewKVTask(item)
	err := task.Init()
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = task.Flush()
		if err != nil {
			t.Fatal(err)
		}
	}()

	err = task.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = task.Stop()
	}()

	{
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
	}

	{
		err = task.InsertStat(&metrics.Stat{
			ServerId: 2,
			Keys:     []string{"127.0.0.2"},
			Hash:     "",
			Value:    3,
			Time:     item.CurrentTime(),
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	{
		err = task.InsertStat(&metrics.Stat{
			ServerId: 1,
			Keys:     []string{"127.0.0.3"},
			Hash:     "",
			Value:    2,
			Time:     item.CurrentTime(),
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	TestKVTask_TestInspect(t)
}

func TestKVTask_CleanExpired(t *testing.T) {
	var task = metrics.NewKVTask(&serverconfigs.MetricItemConfig{
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

	defer func() {
		_ = task.Flush()
	}()

	t.Log("=== inspect ===")
	task.TestInspect(t)
}

func TestKVTask_Upload(t *testing.T) {
	var task = metrics.NewKVTask(&serverconfigs.MetricItemConfig{
		Id:         31,
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

func TestKVTask_TestInspect(t *testing.T) {
	var task = metrics.NewKVTask(&serverconfigs.MetricItemConfig{
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

	task.TestInspect(t)
}

func TestKVTask_Truncate(t *testing.T) {
	var task = metrics.NewKVTask(&serverconfigs.MetricItemConfig{
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

	if testutils.IsSingleTesting() {
		_ = task.Truncate()
	}
}

var testingTask metrics.Task
var testingTaskInitOnce = &sync.Once{}

func initTestingTask() {
	testingTask = metrics.NewKVTask(&serverconfigs.MetricItemConfig{
		Id:         1,
		IsOn:       false,
		Category:   "tcp",
		Period:     1,
		PeriodUnit: serverconfigs.MetricItemPeriodUnitDay,
		Keys:       []string{"${remoteAddr}"},
		Value:      "${countRequest}",
	})

	err := testingTask.Init()
	if err != nil {
		log.Fatal(err)
	}

	err = testingTask.Start()
	if err != nil {
		log.Fatal(err)
	}
}

func BenchmarkKVTask_Add(b *testing.B) {
	runtime.GOMAXPROCS(1)

	testingTaskInitOnce.Do(func() {
		initTestingTask()
	})

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			testingTask.Add(&taskRequest{})
		}
	})
}

type taskRequest struct {
}

func (this *taskRequest) MetricKey(key string) string {
	return configutils.ParseVariables(key, func(varName string) (value string) {
		return "1.2.3.4"
	})
}

func (this *taskRequest) MetricValue(value string) (result int64, ok bool) {
	return 1, true
}

func (this *taskRequest) MetricServerId() int64 {
	return 1
}

func (this *taskRequest) MetricCategory() string {
	return "http"
}

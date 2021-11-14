// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package trackers

import "time"

type tracker struct {
	label     string
	startTime time.Time
}

func Begin(label string) *tracker {
	return &tracker{label: label, startTime: time.Now()}
}

func Run(label string, f func()) {
	var tr = Begin(label)
	f()
	tr.End()
}

func (this *tracker) End() {
	SharedManager.Add(this.label, time.Since(this.startTime).Seconds()*1000)
}

func (this *tracker) Begin(subLabel string) *tracker {
	return Begin(this.label + ":" + subLabel)
}

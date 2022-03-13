// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package dbs

import "time"

type QueryLabel struct {
	manager *QueryStatManager
	query   string
	before  time.Time
}

func NewQueryLabel(manager *QueryStatManager, query string) *QueryLabel {
	return &QueryLabel{
		manager: manager,
		query:   query,
		before:  time.Now(),
	}
}

func (this *QueryLabel) End() {
	var cost = time.Since(this.before).Seconds()
	this.manager.AddCost(this.query, cost)
}

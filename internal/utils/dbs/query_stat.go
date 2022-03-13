// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package dbs

type QueryStat struct {
	Query   string
	CostMin float64
	CostMax float64

	CostTotal float64
	Calls     int64
}

func NewQueryStat(query string) *QueryStat {
	return &QueryStat{
		Query: query,
	}
}

func (this *QueryStat) AddCost(cost float64) {
	if this.CostMin == 0 || this.CostMin > cost {
		this.CostMin = cost
	}
	if this.CostMax == 0 || this.CostMax < cost {
		this.CostMax = cost
	}

	this.CostTotal += cost
	this.Calls++
}

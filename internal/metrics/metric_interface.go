// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package metrics

type MetricInterface interface {
	// MetricKey 指标对象
	MetricKey(key string) string

	// MetricValue 指标值
	MetricValue(value string) (result int64, ok bool)

	// MetricServerId 服务ID
	MetricServerId() int64

	// MetricCategory 指标分类
	MetricCategory() string
}

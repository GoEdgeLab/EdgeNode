package shared

import "time"

type TimeDurationUnit = string

const (
	TimeDurationUnitMS     TimeDurationUnit = "ms"
	TimeDurationUnitSecond TimeDurationUnit = "second"
	TimeDurationUnitMinute TimeDurationUnit = "minute"
	TimeDurationUnitHour   TimeDurationUnit = "hour"
	TimeDurationUnitDay    TimeDurationUnit = "day"
)

// 时间间隔
type TimeDuration struct {
	Count int64            `yaml:"count" json:"count"` // 数量
	Unit  TimeDurationUnit `yaml:"unit" json:"unit"`   // 单位
}

func (this *TimeDuration) Duration() time.Duration {
	switch this.Unit {
	case TimeDurationUnitMS:
		return time.Duration(this.Count) * time.Millisecond
	case TimeDurationUnitSecond:
		return time.Duration(this.Count) * time.Second
	case TimeDurationUnitMinute:
		return time.Duration(this.Count) * time.Minute
	case TimeDurationUnitHour:
		return time.Duration(this.Count) * time.Hour
	case TimeDurationUnitDay:
		return time.Duration(this.Count) * 24 * time.Hour
	default:
		return time.Duration(this.Count) * time.Second
	}
}

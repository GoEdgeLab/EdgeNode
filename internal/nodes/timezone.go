// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package nodes

import (
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"time"
)

func init() {
	// 管理时区
	var lastTimeZone = ""

	events.On(events.EventReload, func() {
		if sharedNodeConfig != nil {
			var timeZone = sharedNodeConfig.TimeZone
			if len(timeZone) == 0 {
				timeZone = "Asia/Shanghai"
			}

			if lastTimeZone != timeZone {
				location, err := time.LoadLocation(timeZone)
				if err != nil {
					remotelogs.Error("TIMEZONE", "change time zone failed: "+err.Error())
					return
				}

				remotelogs.Println("TIMEZONE", "change time zone to '"+timeZone+"'")
				time.Local = location
				lastTimeZone = timeZone
			}
		}
	})
}

// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package stats

import "github.com/mssola/user_agent"

type UserAgentParserResult struct {
	OS             user_agent.OSInfo
	BrowserName    string
	BrowserVersion string
	IsMobile       bool
}

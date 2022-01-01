// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package stats

import "github.com/mssola/user_agent"

type UserAgentParserResult struct {
	os             user_agent.OSInfo
	browserName    string
	browserVersion string
}

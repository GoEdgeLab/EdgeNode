package configutils

import "github.com/iwind/TeaGo/logs"

// 记录错误
func LogError(arg ...interface{}) {
	if len(arg) == 0 {
		return
	}
	logs.Println(arg...)
}

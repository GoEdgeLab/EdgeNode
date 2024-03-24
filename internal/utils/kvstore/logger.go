// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package kvstore

type Logger struct {
}

func NewLogger() *Logger {
	return &Logger{}
}

func (this *Logger) Infof(format string, args ...any) {

}
func (this *Logger) Fatalf(format string, args ...any) {

}

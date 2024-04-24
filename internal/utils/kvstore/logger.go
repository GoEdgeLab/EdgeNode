// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package kvstore

import (
	"fmt"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
)

type Logger struct {
}

func NewLogger() *Logger {
	return &Logger{}
}

func (this *Logger) Infof(format string, args ...any) {
	// stub
}

func (this *Logger) Errorf(format string, args ...any) {
	remotelogs.Error("KV", fmt.Sprintf(format, args...))
}

func (this *Logger) Fatalf(format string, args ...any) {
	remotelogs.Error("KV", fmt.Sprintf(format, args...))
}

// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package writers

import (
	"io"
	"log"
)

type PrintWriter struct {
	rawWriter io.Writer
	tag       string
}

func NewPrintWriter(rawWriter io.Writer, tag string) io.Writer {
	return &PrintWriter{
		rawWriter: rawWriter,
		tag:       tag,
	}
}

func (this *PrintWriter) Write(p []byte) (n int, err error) {
	n, err = this.rawWriter.Write(p)
	if n > 0 {
		log.Println("[" + this.tag + "]" + string(p[:n]))
	}
	return
}

// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package readers

import (
	"bytes"
	"github.com/iwind/TeaGo/types"
	"io"
	"mime/multipart"
	"net/textproto"
	"regexp"
	"strings"
)

type OnPartReadHandler func(start int64, end int64, total int64, data []byte, header textproto.MIMEHeader)

var contentRangeRegexp = regexp.MustCompile(`^(\d+)-(\d+)/(\d+|\*)`)

type ByteRangesReaderCloser struct {
	BaseReader

	rawReader io.ReadCloser
	boundary  string

	mReader *multipart.Reader
	part    *multipart.Part

	buf   *bytes.Buffer
	isEOF bool

	onPartReadHandler OnPartReadHandler
	rangeStart        int64
	rangeEnd          int64
	total             int64

	isStarted bool
	nl        string
}

func NewByteRangesReaderCloser(reader io.ReadCloser, boundary string) *ByteRangesReaderCloser {
	return &ByteRangesReaderCloser{
		rawReader: reader,
		mReader:   multipart.NewReader(reader, boundary),
		boundary:  boundary,
		buf:       &bytes.Buffer{},
		nl:        "\r\n",
	}
}

func (this *ByteRangesReaderCloser) Read(p []byte) (n int, err error) {
	n, err = this.read(p)
	return
}

func (this *ByteRangesReaderCloser) Close() error {
	return this.rawReader.Close()
}

func (this *ByteRangesReaderCloser) OnPartRead(handler OnPartReadHandler) {
	this.onPartReadHandler = handler
}

func (this *ByteRangesReaderCloser) read(p []byte) (n int, err error) {
	// read from buffer
	n, err = this.buf.Read(p)
	if !this.isEOF {
		err = nil
	}
	if n > 0 {
		return
	}
	if this.isEOF {
		return
	}

	if this.part == nil {
		part, partErr := this.mReader.NextPart()
		if partErr != nil {
			if partErr == io.EOF {
				this.buf.WriteString(this.nl + "--" + this.boundary + "--" + this.nl)
				this.isEOF = true
				n, _ = this.buf.Read(p)
				return
			}

			return 0, partErr
		}

		if !this.isStarted {
			this.isStarted = true
			this.buf.WriteString("--" + this.boundary + this.nl)
		} else {
			this.buf.WriteString(this.nl + "--" + this.boundary + this.nl)
		}

		// Headers
		var hasRange = false
		for k, v := range part.Header {
			for _, v1 := range v {
				this.buf.WriteString(k + ": " + v1 + this.nl)

				// parse range
				if k == "Content-Range" {
					var bytesPrefix = "bytes "
					if strings.HasPrefix(v1, bytesPrefix) {
						var r = v1[len(bytesPrefix):]
						var matches = contentRangeRegexp.FindStringSubmatch(r)
						if len(matches) > 2 {
							var start = types.Int64(matches[1])
							var end = types.Int64(matches[2])
							var total int64 = 0
							if matches[3] != "*" {
								total = types.Int64(matches[3])
							}
							if start <= end {
								hasRange = true
								this.rangeStart = start
								this.rangeEnd = end
								this.total = total
							}
						}
					}
				}
			}
		}

		if !hasRange {
			this.rangeStart = -1
			this.rangeEnd = -1
		}

		this.buf.WriteString(this.nl)
		this.part = part

		n, _ = this.buf.Read(p)
		return
	}

	n, err = this.part.Read(p)

	if this.onPartReadHandler != nil && n > 0 && this.rangeStart >= 0 && this.rangeEnd >= 0 {
		this.onPartReadHandler(this.rangeStart, this.rangeEnd, this.total, p[:n], this.part.Header)
		this.rangeStart += int64(n)
	}

	if err == io.EOF {
		this.part = nil
		err = nil

		// 如果没有读取到内容，则直接跳到下一个Part
		if n == 0 {
			return this.read(p)
		}
	}

	return
}

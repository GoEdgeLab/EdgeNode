// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package utils

import "io"

func CopyWithFilter(writer io.Writer, reader io.Reader, buf []byte, filter func(p []byte) []byte) (written int64, err error) {
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			n2, err := writer.Write(filter(buf[:n]))
			written += int64(n2)
			if err != nil {
				return written, err
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return written, err
		}
	}
	return written, nil
}

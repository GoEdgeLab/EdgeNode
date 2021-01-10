package nodes

import (
	"crypto/rand"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// 分解Range
func httpRequestParseContentRange(rangeValue string) (result [][]int, ok bool) {
	// 参考RFC：https://tools.ietf.org/html/rfc7233
	index := strings.Index(rangeValue, "=")
	if index == -1 {
		return
	}
	unit := rangeValue[:index]
	if unit != "bytes" {
		return
	}

	rangeSetString := rangeValue[index+1:]
	if len(rangeSetString) == 0 {
		ok = true
		return
	}

	pieces := strings.Split(rangeSetString, ", ")
	for _, piece := range pieces {
		index := strings.Index(piece, "-")
		if index == -1 {
			return
		}
		first := piece[:index]
		firstInt := -1

		var err error
		last := piece[index+1:]
		var lastInt = -1

		if len(first) > 0 {
			firstInt, err = strconv.Atoi(first)
			if err != nil {
				return
			}

			if len(last) > 0 {
				lastInt, err = strconv.Atoi(last)
				if err != nil {
					return
				}
				if lastInt < firstInt {
					return
				}
			}
		} else {
			if len(last) == 0 {
				return
			}

			lastInt, err = strconv.Atoi(last)
			if err != nil {
				return
			}
			lastInt = -lastInt
		}

		result = append(result, []int{firstInt, lastInt})
	}

	ok = true
	return
}

// 读取内容Range
func httpRequestReadRange(reader io.Reader, buf []byte, start int, end int, callback func(buf []byte, n int) error) (ok bool, err error) {
	if start < 0 || end < 0 {
		return
	}
	seeker, ok := reader.(io.Seeker)
	if !ok {
		return
	}
	_, err = seeker.Seek(int64(start), io.SeekStart)
	if err != nil {
		return false, nil
	}

	offset := start
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			offset += n
			if end < offset {
				err = callback(buf, n-(offset-end-1))
				if err != nil {
					return false, err
				}
				return true, nil
			} else {
				err = callback(buf, n)
				if err != nil {
					return false, err
				}
			}
		}

		if err != nil {
			if err == io.EOF {
				return true, nil
			}
			return false, err
		}
	}
}

// 生成boundary
// 仿照Golang自带的函数（multipart包）
func httpRequestGenBoundary() string {
	var buf [30]byte
	_, err := io.ReadFull(rand.Reader, buf[:])
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", buf[:])
}

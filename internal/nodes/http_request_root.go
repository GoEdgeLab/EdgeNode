package nodes

import (
	"fmt"
	"github.com/cespare/xxhash"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/logs"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// 文本mime-type列表
var textMimeMap = map[string]bool{
	"application/atom+xml":                true,
	"application/javascript":              true,
	"application/x-javascript":            true,
	"application/json":                    true,
	"application/rss+xml":                 true,
	"application/x-web-app-manifest+json": true,
	"application/xhtml+xml":               true,
	"application/xml":                     true,
	"image/svg+xml":                       true,
	"text/css":                            true,
	"text/plain":                          true,
	"text/javascript":                     true,
	"text/xml":                            true,
	"text/html":                           true,
	"text/xhtml":                          true,
	"text/sgml":                           true,
}

// 调用本地静态资源
// 如果返回true，则终止请求
func (this *HTTPRequest) doRoot() (isBreak bool) {
	if this.web.Root == nil || !this.web.Root.IsOn {
		return
	}

	if len(this.uri) == 0 {
		this.write404()
		return true
	}

	rootDir := this.web.Root.Dir
	if this.web.Root.HasVariables() {
		rootDir = this.Format(rootDir)
	}
	if !filepath.IsAbs(rootDir) {
		rootDir = Tea.Root + Tea.DS + rootDir
	}

	requestPath := this.uri

	questionMarkIndex := strings.Index(this.uri, "?")
	if questionMarkIndex > -1 {
		requestPath = this.uri[:questionMarkIndex]
	}

	// 去掉其中的奇怪的路径
	requestPath = strings.Replace(requestPath, "..\\", "", -1)

	// 进行URL Decode
	if this.web.Root.DecodePath {
		p, err := url.QueryUnescape(requestPath)
		if err == nil {
			requestPath = p
		} else {
			logs.Error(err)
		}
	}

	// 去掉前缀
	stripPrefix := this.web.Root.StripPrefix
	if len(stripPrefix) > 0 {
		if stripPrefix[0] != '/' {
			stripPrefix = "/" + stripPrefix
		}

		requestPath = strings.TrimPrefix(requestPath, stripPrefix)
		if len(requestPath) == 0 || requestPath[0] != '/' {
			requestPath = "/" + requestPath
		}
	}

	filename := strings.Replace(requestPath, "/", Tea.DS, -1)
	filePath := ""
	if len(filename) > 0 && filename[0:1] == Tea.DS {
		filePath = rootDir + filename
	} else {
		filePath = rootDir + Tea.DS + filename
	}

	this.filePath = filePath // 用来记录日志

	stat, err := os.Stat(filePath)
	if err != nil {
		_, isPathError := err.(*fs.PathError)
		if os.IsNotExist(err) || isPathError {
			if this.web.Root.IsBreak {
				this.write404()
				return true
			}
			return
		} else {
			this.write500(err)
			logs.Error(err)
			return true
		}
	}
	if stat.IsDir() {
		indexFile, indexStat := this.findIndexFile(filePath)
		if len(indexFile) > 0 {
			filePath += Tea.DS + indexFile
		} else {
			if this.web.Root.IsBreak {
				this.write404()
				return true
			}
			return
		}
		this.filePath = filePath

		// stat again
		if indexStat == nil {
			stat, err = os.Stat(filePath)
			if err != nil {
				if os.IsNotExist(err) {
					if this.web.Root.IsBreak {
						this.write404()
						return true
					}
					return
				} else {
					this.write500(err)
					logs.Error(err)
					return true
				}
			}
		} else {
			stat = indexStat
		}
	}

	// 响应header
	respHeader := this.writer.Header()

	// mime type
	contentType := ""
	if this.web.ResponseHeaderPolicy == nil || !this.web.ResponseHeaderPolicy.IsOn || !this.web.ResponseHeaderPolicy.ContainsHeader("CONTENT-TYPE") {
		ext := filepath.Ext(filePath)
		if len(ext) > 0 {
			mimeType := mime.TypeByExtension(ext)
			if len(mimeType) > 0 {
				semicolonIndex := strings.Index(mimeType, ";")
				mimeTypeKey := mimeType
				if semicolonIndex > 0 {
					mimeTypeKey = mimeType[:semicolonIndex]
				}

				if _, found := textMimeMap[mimeTypeKey]; found {
					if this.web.Charset != nil && this.web.Charset.IsOn && len(this.web.Charset.Charset) > 0 {
						charset := this.web.Charset.Charset
						if this.web.Charset.IsUpper {
							charset = strings.ToUpper(charset)
						}
						contentType = mimeTypeKey + "; charset=" + charset
						respHeader.Set("Content-Type", mimeTypeKey+"; charset="+charset)
					} else {
						contentType = mimeType
						respHeader.Set("Content-Type", mimeType)
					}
				} else {
					contentType = mimeType
					respHeader.Set("Content-Type", mimeType)
				}
			}
		}
	}

	// length
	fileSize := stat.Size()

	// 支持 Last-Modified
	modifiedTime := stat.ModTime().Format("Mon, 02 Jan 2006 15:04:05 GMT")
	if len(respHeader.Get("Last-Modified")) == 0 {
		respHeader.Set("Last-Modified", modifiedTime)
	}

	// 支持 ETag
	eTag := "\"e" + fmt.Sprintf("%0x", xxhash.Sum64String(filename+strconv.FormatInt(stat.ModTime().UnixNano(), 10)+strconv.FormatInt(stat.Size(), 10))) + "\""
	if len(respHeader.Get("ETag")) == 0 {
		respHeader.Set("ETag", eTag)
	}

	// 支持 If-None-Match
	if this.requestHeader("If-None-Match") == eTag {
		// 自定义Header
		this.processResponseHeaders(http.StatusNotModified)
		this.writer.WriteHeader(http.StatusNotModified)
		return true
	}

	// 支持 If-Modified-Since
	if this.requestHeader("If-Modified-Since") == modifiedTime {
		// 自定义Header
		this.processResponseHeaders(http.StatusNotModified)
		this.writer.WriteHeader(http.StatusNotModified)
		return true
	}

	// 支持Range
	respHeader.Set("Accept-Ranges", "bytes")
	ifRangeHeaders, ok := this.RawReq.Header["If-Range"]
	supportRange := true
	if ok {
		supportRange = false
		for _, v := range ifRangeHeaders {
			if v == eTag || v == modifiedTime {
				supportRange = true
			}
		}
		if !supportRange {
			respHeader.Del("Accept-Ranges")
		}
	}

	// 支持Range
	rangeSet := [][]int64{}
	if supportRange {
		contentRange := this.RawReq.Header.Get("Range")
		if len(contentRange) > 0 {
			if fileSize == 0 {
				this.processResponseHeaders(http.StatusRequestedRangeNotSatisfiable)
				this.writer.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
				return true
			}

			set, ok := httpRequestParseContentRange(contentRange)
			if !ok {
				this.processResponseHeaders(http.StatusRequestedRangeNotSatisfiable)
				this.writer.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
				return true
			}
			if len(set) > 0 {
				rangeSet = set
				for _, arr := range rangeSet {
					if arr[0] == -1 {
						arr[0] = fileSize + arr[1]
						arr[1] = fileSize - 1

						if arr[0] < 0 {
							this.processResponseHeaders(http.StatusRequestedRangeNotSatisfiable)
							this.writer.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
							return true
						}
					}
					if arr[1] > 0 {
						arr[1] = fileSize - 1
					}
					if arr[1] < 0 {
						arr[1] = fileSize - 1
					}
					if arr[0] > arr[1] {
						this.processResponseHeaders(http.StatusRequestedRangeNotSatisfiable)
						this.writer.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
						return true
					}
				}
			}
		} else {
			respHeader.Set("Content-Length", strconv.FormatInt(fileSize, 10))
		}
	} else {
		respHeader.Set("Content-Length", strconv.FormatInt(fileSize, 10))
	}

	reader, err := os.OpenFile(filePath, os.O_RDONLY, 0444)
	if err != nil {
		this.write500(err)
		logs.Error(err)
		return true
	}

	// 自定义Header
	this.processResponseHeaders(http.StatusOK)

	// 在Range请求中不能缓存
	if len(rangeSet) > 0 {
		this.cacheRef = nil // 不支持缓存
	}

	this.writer.Prepare(fileSize, http.StatusOK)

	pool := this.bytePool(fileSize)
	buf := pool.Get()
	defer func() {
		_ = reader.Close()
		pool.Put(buf)
	}()

	if len(rangeSet) == 1 {
		respHeader.Set("Content-Range", "bytes "+strconv.FormatInt(rangeSet[0][0], 10)+"-"+strconv.FormatInt(rangeSet[0][1], 10)+"/"+strconv.FormatInt(fileSize, 10))
		this.writer.WriteHeader(http.StatusPartialContent)

		ok, err := httpRequestReadRange(reader, buf, rangeSet[0][0], rangeSet[0][1], func(buf []byte, n int) error {
			_, err := this.writer.Write(buf[:n])
			return err
		})
		if err != nil {
			logs.Error(err)
			return true
		}
		if !ok {
			this.processResponseHeaders(http.StatusRequestedRangeNotSatisfiable)
			this.writer.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return true
		}
	} else if len(rangeSet) > 1 {
		boundary := httpRequestGenBoundary()
		respHeader.Set("Content-Type", "multipart/byteranges; boundary="+boundary)

		this.writer.WriteHeader(http.StatusPartialContent)

		for index, set := range rangeSet {
			if index == 0 {
				_, err = this.writer.WriteString("--" + boundary + "\r\n")
			} else {
				_, err = this.writer.WriteString("\r\n--" + boundary + "\r\n")
			}
			if err != nil {
				logs.Error(err)
				return true
			}

			_, err = this.writer.WriteString("Content-Range: " + "bytes " + strconv.FormatInt(set[0], 10) + "-" + strconv.FormatInt(set[1], 10) + "/" + strconv.FormatInt(fileSize, 10) + "\r\n")
			if err != nil {
				logs.Error(err)
				return true
			}

			if len(contentType) > 0 {
				_, err = this.writer.WriteString("Content-Type: " + contentType + "\r\n\r\n")
				if err != nil {
					logs.Error(err)
					return true
				}
			}

			ok, err := httpRequestReadRange(reader, buf, set[0], set[1], func(buf []byte, n int) error {
				_, err := this.writer.Write(buf[:n])
				return err
			})
			if err != nil {
				logs.Error(err)
				return true
			}
			if !ok {
				this.processResponseHeaders(http.StatusRequestedRangeNotSatisfiable)
				this.writer.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
				return true
			}
		}

		_, err = this.writer.WriteString("\r\n--" + boundary + "--\r\n")
		if err != nil {
			logs.Error(err)
			return true
		}
	} else {
		_, err = io.CopyBuffer(this.writer, reader, buf)

		if err != nil {
			logs.Error(err)
			return true
		}
	}

	// 设置成功
	this.writer.SetOk()

	return true
}

// 查找首页文件
func (this *HTTPRequest) findIndexFile(dir string) (filename string, stat os.FileInfo) {
	if this.web.Root == nil || !this.web.Root.IsOn {
		return "", nil
	}
	if len(this.web.Root.Indexes) == 0 {
		return "", nil
	}
	for _, index := range this.web.Root.Indexes {
		if len(index) == 0 {
			continue
		}

		// 模糊查找
		if strings.Contains(index, "*") {
			indexFiles, err := filepath.Glob(dir + Tea.DS + index)
			if err != nil {
				logs.Error(err)
				this.addError(err)
				continue
			}
			if len(indexFiles) > 0 {
				return filepath.Base(indexFiles[0]), nil
			}
			continue
		}

		// 精确查找
		filePath := dir + Tea.DS + index
		stat, err := os.Stat(filePath)
		if err != nil || !stat.Mode().IsRegular() {
			continue
		}
		return index, stat
	}
	return "", nil
}

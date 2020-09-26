package nodes

import (
	"fmt"
	"github.com/dchest/siphash"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/logs"
	"io"
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
		if os.IsNotExist(err) {
			if this.web.Root.IsBreak {
				this.write404()
				return true
			}
			return
		} else {
			this.write500()
			logs.Error(err)
			this.addError(err)
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
					this.write500()
					logs.Error(err)
					this.addError(err)
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
	if this.web.ResponseHeaderPolicy == nil || !this.web.ResponseHeaderPolicy.IsOn || !this.web.ResponseHeaderPolicy.ContainsHeader("CONTENT-TYPE") {
		ext := filepath.Ext(requestPath)
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
						respHeader.Set("Content-Type", mimeTypeKey+"; charset="+charset)
					} else {
						respHeader.Set("Content-Type", mimeType)
					}
				} else {
					respHeader.Set("Content-Type", mimeType)
				}
			}
		}
	}

	// length
	fileSize := stat.Size()
	respHeader.Set("Content-Length", strconv.FormatInt(fileSize, 10))

	// 支持 Last-Modified
	modifiedTime := stat.ModTime().Format("Mon, 02 Jan 2006 15:04:05 GMT")
	if len(respHeader.Get("Last-Modified")) == 0 {
		respHeader.Set("Last-Modified", modifiedTime)
	}

	// 支持 ETag
	eTag := "\"et" + fmt.Sprintf("%0x", siphash.Hash(0, 0, []byte(filename+strconv.FormatInt(stat.ModTime().UnixNano(), 10)+strconv.FormatInt(stat.Size(), 10)))) + "\""
	if len(respHeader.Get("ETag")) == 0 {
		respHeader.Set("ETag", eTag)
	}

	// proxy callback
	// TODO

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

	// 自定义Header
	this.processResponseHeaders(http.StatusOK)

	reader, err := os.OpenFile(filePath, os.O_RDONLY, 0444)
	if err != nil {
		this.write500()
		logs.Error(err)
		this.addError(err)
		return true
	}

	this.writer.Prepare(fileSize)

	pool := this.bytePool(fileSize)
	buf := pool.Get()
	_, err = io.CopyBuffer(this.writer, reader, buf)
	pool.Put(buf)

	// 不使用defer，以便于加快速度
	_ = reader.Close()

	if err != nil {
		logs.Error(err)
		return true
	}

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

package checkpoints

import (
	"bytes"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/TeaOSLab/EdgeNode/internal/waf/utils"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var multipartHeaderRegexp = regexp.MustCompile(`(?i)(?:^|\r\n)--+\w+\r\n((([\w-]+: .+)\r\n)+)`)
var multipartHeaderContentRangeRegexp = regexp.MustCompile(`/(\d+)`)

// RequestUploadCheckpoint ${requestUpload.arg}
type RequestUploadCheckpoint struct {
	Checkpoint
}

func (this *RequestUploadCheckpoint) RequestValue(req requests.Request, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	if this.RequestBodyIsEmpty(req) {
		value = ""
		return
	}

	value = ""
	if param == "minSize" || param == "maxSize" {
		value = 0
	}

	if req.WAFRaw().Method != http.MethodPost {
		return
	}

	if req.WAFRaw().Body == nil {
		return
	}

	hasRequestBody = true

	var requestContentLength = req.WAFRaw().ContentLength

	var fields []string
	var minSize int64
	var maxSize int64
	var names []string
	var extensions []string

	if requestContentLength <= req.WAFMaxRequestSize() { // full read
		if req.WAFRaw().MultipartForm == nil {
			var bodyData = req.WAFGetCacheBody()
			if len(bodyData) == 0 {
				data, err := req.WAFReadBody(req.WAFMaxRequestSize())
				if err != nil {
					sysErr = err
					return
				}

				bodyData = data
				req.WAFSetCacheBody(data)
				defer req.WAFRestoreBody(data)
			}
			var oldBody = req.WAFRaw().Body
			req.WAFRaw().Body = io.NopCloser(bytes.NewBuffer(bodyData))
			err := req.WAFRaw().ParseMultipartForm(req.WAFMaxRequestSize())
			if err == nil {
				for field, files := range req.WAFRaw().MultipartForm.File {
					if param == "field" {
						fields = append(fields, field)
					} else if param == "minSize" {
						for _, file := range files {
							if minSize == 0 || minSize > file.Size {
								minSize = file.Size
							}
						}
					} else if param == "maxSize" {
						for _, file := range files {
							if maxSize < file.Size {
								maxSize = file.Size
							}
						}
					} else if param == "name" {
						for _, file := range files {
							if !lists.ContainsString(names, file.Filename) {
								names = append(names, file.Filename)
							}
						}
					} else if param == "ext" {
						for _, file := range files {
							if len(file.Filename) > 0 {
								exit := strings.ToLower(filepath.Ext(file.Filename))
								if !lists.ContainsString(extensions, exit) {
									extensions = append(extensions, exit)
								}
							}
						}
					}
				}
			}

			// 还原
			req.WAFRaw().Body = oldBody

			if err != nil {
				userErr = err
				return
			}

			if req.WAFRaw().MultipartForm == nil {
				return
			}
		}
	} else { // read first part
		var bodyData = req.WAFGetCacheBody()
		if len(bodyData) == 0 {
			data, err := req.WAFReadBody(req.WAFMaxRequestSize())
			if err != nil {
				sysErr = err
				return
			}

			bodyData = data
			req.WAFSetCacheBody(data)
			defer req.WAFRestoreBody(data)
		}

		var subMatches = multipartHeaderRegexp.FindAllSubmatch(bodyData, -1)
		for _, subMatch := range subMatches {
			var headers = bytes.Split(subMatch[1], []byte{'\r', '\n'})
			var partContentLength int64 = -1
			for _, header := range headers {
				if len(header) > 2 {
					var kv = bytes.SplitN(header, []byte{':'}, 2)
					if len(kv) == 2 {
						var k = kv[0]
						var v = kv[1]
						switch string(bytes.ToLower(k)) {
						case "content-disposition":
							var props = bytes.Split(v, []byte{';', ' '})
							for _, prop := range props {
								var propKV = bytes.SplitN(prop, []byte{'='}, 2)
								if len(propKV) == 2 {
									var propValue = string(propKV[1])
									switch string(propKV[0]) {
									case "name":
										if param == "field" {
											propValue, _ = strconv.Unquote(propValue)
											fields = append(fields, propValue)
										}
									case "filename":
										if param == "name" {
											propValue, _ = strconv.Unquote(propValue)
											names = append(names, propValue)
										} else if param == "ext" {
											propValue, _ = strconv.Unquote(propValue)
											extensions = append(extensions, strings.ToLower(filepath.Ext(propValue)))
										}
									}
								}
							}
						case "content-range":
							if partContentLength <= 0 {
								var contentRange = multipartHeaderContentRangeRegexp.FindSubmatch(v)
								if len(contentRange) >= 2 {
									partContentLength = types.Int64(string(contentRange[1]))
								}
							}
						case "content-length":
							if partContentLength <= 0 {
								partContentLength = types.Int64(string(v))
							}
						}
					}
				}
			}

			// minSize & maxSize
			if partContentLength > 0 {
				if param == "minSize" && (minSize == 0 /** not set yet **/ || partContentLength < minSize) {
					minSize = partContentLength
				} else if param == "maxSize" && partContentLength > maxSize {
					maxSize = partContentLength
				}
			}
		}
	}

	if param == "field" { // field
		value = strings.Join(fields, ",")
	} else if param == "minSize" { // minSize
		if minSize == 0 && requestContentLength > 0 {
			minSize = requestContentLength
		}
		value = minSize
	} else if param == "maxSize" { // maxSize
		if maxSize == 0 && requestContentLength > 0 {
			maxSize = requestContentLength
		}
		value = maxSize
	} else if param == "name" { // name
		value = strings.Join(names, ",")
	} else if param == "ext" { // ext
		value = strings.Join(extensions, ",")
	}

	return
}

func (this *RequestUploadCheckpoint) ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options, ruleId)
	}
	return
}

func (this *RequestUploadCheckpoint) ParamOptions() *ParamOptions {
	option := NewParamOptions()
	option.AddParam("最小文件尺寸", "minSize")
	option.AddParam("最大文件尺寸", "maxSize")
	option.AddParam("扩展名(如.txt)", "ext")
	option.AddParam("原始文件名", "name")
	option.AddParam("表单字段名", "field")
	return option
}

func (this *RequestUploadCheckpoint) CacheLife() utils.CacheLife {
	return utils.CacheMiddleLife
}

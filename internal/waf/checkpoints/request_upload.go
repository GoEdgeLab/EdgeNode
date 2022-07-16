package checkpoints

import (
	"bytes"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/TeaOSLab/EdgeNode/internal/waf/utils"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/maps"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
)

// RequestUploadCheckpoint ${requestUpload.arg}
type RequestUploadCheckpoint struct {
	Checkpoint
}

func (this *RequestUploadCheckpoint) RequestValue(req requests.Request, param string, options maps.Map) (value interface{}, hasRequestBody bool, sysErr error, userErr error) {
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
	if req.WAFRaw().MultipartForm == nil {
		var bodyData = req.WAFGetCacheBody()
		if len(bodyData) == 0 {
			data, err := req.WAFReadBody(utils.MaxBodySize)
			if err != nil {
				sysErr = err
				return
			}

			bodyData = data
			req.WAFSetCacheBody(data)
			defer req.WAFRestoreBody(data)
		}
		oldBody := req.WAFRaw().Body
		req.WAFRaw().Body = ioutil.NopCloser(bytes.NewBuffer(bodyData))

		err := req.WAFRaw().ParseMultipartForm(utils.MaxBodySize)

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

	if param == "field" { // field
		fields := []string{}
		for field := range req.WAFRaw().MultipartForm.File {
			fields = append(fields, field)
		}
		value = strings.Join(fields, ",")
	} else if param == "minSize" { // minSize
		minSize := int64(0)
		for _, files := range req.WAFRaw().MultipartForm.File {
			for _, file := range files {
				if minSize == 0 || minSize > file.Size {
					minSize = file.Size
				}
			}
		}
		value = minSize
	} else if param == "maxSize" { // maxSize
		maxSize := int64(0)
		for _, files := range req.WAFRaw().MultipartForm.File {
			for _, file := range files {
				if maxSize < file.Size {
					maxSize = file.Size
				}
			}
		}
		value = maxSize
	} else if param == "name" { // name
		names := []string{}
		for _, files := range req.WAFRaw().MultipartForm.File {
			for _, file := range files {
				if !lists.ContainsString(names, file.Filename) {
					names = append(names, file.Filename)
				}
			}
		}
		value = strings.Join(names, ",")
	} else if param == "ext" { // ext
		extensions := []string{}
		for _, files := range req.WAFRaw().MultipartForm.File {
			for _, file := range files {
				if len(file.Filename) > 0 {
					exit := strings.ToLower(filepath.Ext(file.Filename))
					if !lists.ContainsString(extensions, exit) {
						extensions = append(extensions, exit)
					}
				}
			}
		}
		value = strings.Join(extensions, ",")
	}

	return
}

func (this *RequestUploadCheckpoint) ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map) (value interface{}, hasRequestBody bool, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options)
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

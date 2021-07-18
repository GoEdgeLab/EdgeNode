// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package waf

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"net/http"
	"time"
)

var get302Validator = NewGet302Validator()

type Get302Validator struct {
}

func NewGet302Validator() *Get302Validator {
	return &Get302Validator{}
}

func (this *Get302Validator) Run(request requests.Request, writer http.ResponseWriter) {
	var info = request.WAFRaw().URL.Query().Get("info")
	if len(info) == 0 {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte("invalid request"))
		return
	}
	m, err := utils.SimpleDecryptMap(info)
	if err != nil {
		_, _ = writer.Write([]byte("invalid request"))
		return
	}

	var timestamp = m.GetInt64("timestamp")
	if time.Now().Unix()-timestamp > 5 { // 超过5秒认为失效
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte("invalid request"))
		return
	}

	// 加入白名单
	life := m.GetInt64("life")
	if life <= 0 {
		life = 600 // 默认10分钟
	}
	setId := m.GetString("setId")
	SharedIPWhiteList.Add("set:"+setId, request.WAFRemoteIP(), time.Now().Unix()+life)

	// 返回原始URL
	var url = m.GetString("url")
	http.Redirect(writer, request.WAFRaw(), url, http.StatusFound)
}

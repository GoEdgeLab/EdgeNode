// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package waf

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/types"
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
		request.ProcessResponseHeaders(writer.Header(), http.StatusBadRequest)
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte("invalid request"))
		return
	}
	m, err := utils.SimpleDecryptMap(info)
	if err != nil {
		request.ProcessResponseHeaders(writer.Header(), http.StatusBadRequest)
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte("invalid request"))
		return
	}

	var timestamp = m.GetInt64("timestamp")
	if time.Now().Unix()-timestamp > 5 { // 超过5秒认为失效
		request.ProcessResponseHeaders(writer.Header(), http.StatusBadRequest)
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte("invalid request"))
		return
	}

	// 加入白名单
	var life = m.GetInt64("life")
	if life <= 0 {
		life = 600 // 默认10分钟
	}
	var setId = types.String(m.GetInt64("setId"))
	SharedIPWhiteList.RecordIP("set:"+setId, m.GetString("scope"), request.WAFServerId(), request.WAFRemoteIP(), time.Now().Unix()+life, m.GetInt64("policyId"), false, m.GetInt64("groupId"), m.GetInt64("setId"), "")

	// 返回原始URL
	var url = m.GetString("url")

	request.ProcessResponseHeaders(writer.Header(), http.StatusFound)
	http.Redirect(writer, request.WAFRaw(), url, http.StatusFound)
}

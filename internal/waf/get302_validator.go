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
		_, _ = writer.Write([]byte("invalid request (002)"))
		return
	}

	var timestamp int64
	var life int64
	var setId int64
	var policyId int64
	var groupId int64
	var scope string
	var url string

	var infoArg = &InfoArg{}
	decodeErr := infoArg.Decode(info)
	var success bool
	if decodeErr == nil && infoArg.IsValid() {
		success = true

		timestamp = infoArg.Timestamp
		life = int64(infoArg.Life)
		setId = infoArg.SetId
		policyId = infoArg.PolicyId
		groupId = infoArg.GroupId
		scope = infoArg.Scope
		url = infoArg.URL
	} else {
		// 兼容老版本
		m, decodeMapErr := utils.SimpleDecryptMap(info)
		if decodeMapErr == nil {
			success = true

			timestamp = m.GetInt64("timestamp")
			life = m.GetInt64("life")
			setId = m.GetInt64("setId")
			policyId = m.GetInt64("policyId")
			groupId = m.GetInt64("groupId")
			scope = m.GetString("scope")
			url = m.GetString("url")
		}
	}

	if !success {
		request.ProcessResponseHeaders(writer.Header(), http.StatusBadRequest)
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte("invalid request (003)"))
		return
	}

	if time.Now().Unix()-timestamp > 5 { // 超过5秒认为失效
		request.ProcessResponseHeaders(writer.Header(), http.StatusBadRequest)
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte("invalid request (004)"))
		return
	}

	// 加入白名单
	if life <= 0 {
		life = 600 // 默认10分钟
	}
	SharedIPWhiteList.RecordIP("set:"+types.String(setId), scope, request.WAFServerId(), request.WAFRemoteIP(), time.Now().Unix()+life, policyId, false, groupId, setId, "")

	// 返回原始URL
	request.ProcessResponseHeaders(writer.Header(), http.StatusFound)
	http.Redirect(writer, request.WAFRaw(), url, http.StatusFound)
}

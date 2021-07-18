package waf

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"net/http"
	"strings"
	"time"
)

type recordIPTask struct {
	ip        string
	listId    int64
	expiredAt int64
	level     string
}

var recordIPTaskChan = make(chan *recordIPTask, 1024)

func init() {
	events.On(events.EventLoaded, func() {
		go func() {
			rpcClient, err := rpc.SharedRPC()
			if err != nil {
				remotelogs.Error("WAF_RECORD_IP_ACTION", "create rpc client failed: "+err.Error())
				return
			}

			for task := range recordIPTaskChan {
				ipType := "ipv4"
				if strings.Contains(task.ip, ":") {
					ipType = "ipv6"
				}
				_, err = rpcClient.IPItemRPC().CreateIPItem(rpcClient.Context(), &pb.CreateIPItemRequest{
					IpListId:   task.listId,
					IpFrom:     task.ip,
					IpTo:       "",
					ExpiredAt:  task.expiredAt,
					Reason:     "触发WAF规则自动加入",
					Type:       ipType,
					EventLevel: task.level,
				})
				if err != nil {
					remotelogs.Error("WAF_RECORD_IP_ACTION", "create ip item failed: "+err.Error())
				}
			}
		}()
	})
}

type RecordIPAction struct {
	BaseAction

	Type     string `yaml:"type" json:"type"`
	IPListId int64  `yaml:"ipListId" json:"ipListId"`
	Level    string `yaml:"level" json:"level"`
	Timeout  int32  `yaml:"timeout" json:"timeout"`
}

func (this *RecordIPAction) Init(waf *WAF) error {
	return nil
}

func (this *RecordIPAction) Code() string {
	return ActionRecordIP
}

func (this *RecordIPAction) IsAttack() bool {
	return this.Type == "black"
}

func (this *RecordIPAction) WillChange() bool {
	return this.Type == "black"
}

func (this *RecordIPAction) Perform(waf *WAF, group *RuleGroup, set *RuleSet, request requests.Request, writer http.ResponseWriter) (allow bool) {
	// 是否在本地白名单中
	if SharedIPWhiteList.Contains("set:"+set.Id, set.Id) {
		return true
	}

	// 先加入本地的黑名单
	timeout := this.Timeout
	if timeout <= 0 {
		timeout = 86400 // 1天
	}
	expiredAt := time.Now().Unix() + int64(timeout)

	if this.Type == "black" {
		_ = this.CloseConn(writer)

		SharedIPBlackLIst.Add(IPTypeAll, request.WAFRemoteIP(), expiredAt)
	} else {
		// 加入本地白名单
		timeout := this.Timeout
		if timeout <= 0 {
			timeout = 86400 // 1天
		}
		SharedIPWhiteList.Add("set:"+set.Id, request.WAFRemoteIP(), expiredAt)
	}

	// 上报
	if this.IPListId > 0 {
		select {
		case recordIPTaskChan <- &recordIPTask{
			ip:        request.WAFRemoteIP(),
			listId:    this.IPListId,
			expiredAt: expiredAt,
			level:     this.Level,
		}:
		default:

		}
	}

	return this.Type != "black"
}

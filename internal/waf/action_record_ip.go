package waf

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/types"
	"net/http"
	"strings"
	"time"
)

type recordIPTask struct {
	ip        string
	listId    int64
	expiredAt int64
	level     string
	serverId  int64

	sourceServerId                int64
	sourceHTTPFirewallPolicyId    int64
	sourceHTTPFirewallRuleGroupId int64
	sourceHTTPFirewallRuleSetId   int64
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
					IpListId:                      task.listId,
					IpFrom:                        task.ip,
					IpTo:                          "",
					ExpiredAt:                     task.expiredAt,
					Reason:                        "触发WAF规则自动加入",
					Type:                          ipType,
					EventLevel:                    task.level,
					ServerId:                      task.serverId,
					SourceNodeId:                  teaconst.NodeId,
					SourceServerId:                task.sourceServerId,
					SourceHTTPFirewallPolicyId:    task.sourceHTTPFirewallPolicyId,
					SourceHTTPFirewallRuleGroupId: task.sourceHTTPFirewallRuleGroupId,
					SourceHTTPFirewallRuleSetId:   task.sourceHTTPFirewallRuleSetId,
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
	Scope    string `yaml:"scope" json:"scope"`
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
	if SharedIPWhiteList.Contains("set:"+types.String(set.Id), this.Scope, request.WAFServerId(), request.WAFRemoteIP()) {
		return true
	}

	timeout := this.Timeout
	if timeout <= 0 {
		timeout = 86400 // 1天
	}
	expiredAt := time.Now().Unix() + int64(timeout)

	if this.Type == "black" {
		writer.WriteHeader(http.StatusForbidden)

		request.WAFClose()

		// 先加入本地的黑名单
		SharedIPBlackList.Add(IPTypeAll, this.Scope, request.WAFServerId(), request.WAFRemoteIP(), expiredAt)
	} else {
		// 加入本地白名单
		SharedIPWhiteList.Add("set:"+types.String(set.Id), this.Scope, request.WAFServerId(), request.WAFRemoteIP(), expiredAt)
	}

	// 上报
	if this.IPListId > 0 {
		var serverId int64
		if this.Scope == firewallconfigs.FirewallScopeService {
			serverId = request.WAFServerId()
		}

		select {
		case recordIPTaskChan <- &recordIPTask{
			ip:                            request.WAFRemoteIP(),
			listId:                        this.IPListId,
			expiredAt:                     expiredAt,
			level:                         this.Level,
			serverId:                      serverId,
			sourceServerId:                request.WAFServerId(),
			sourceHTTPFirewallPolicyId:    waf.Id,
			sourceHTTPFirewallRuleGroupId: group.Id,
			sourceHTTPFirewallRuleSetId:   set.Id,
		}:
		default:

		}
	}

	return this.Type != "black"
}

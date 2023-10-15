package waf

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/firewallconfigs"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
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
	expiresAt int64
	level     string
	serverId  int64

	reason string

	sourceServerId                int64
	sourceHTTPFirewallPolicyId    int64
	sourceHTTPFirewallRuleGroupId int64
	sourceHTTPFirewallRuleSetId   int64
}

var recordIPTaskChan = make(chan *recordIPTask, 2048)

func init() {
	if !teaconst.IsMain {
		return
	}

	events.On(events.EventLoaded, func() {
		goman.New(func() {
			rpcClient, err := rpc.SharedRPC()
			if err != nil {
				remotelogs.Error("WAF_RECORD_IP_ACTION", "create rpc client failed: "+err.Error())
				return
			}

			const maxItems = 512 // 每次上传的最大IP数

			for {
				var pbItemMap = map[string]*pb.CreateIPItemsRequest_IPItem{} // ip => IPItem

				func() {
					for {
						select {
						case task := <-recordIPTaskChan:
							var ipType = "ipv4"
							if strings.Contains(task.ip, ":") {
								ipType = "ipv6"
							}
							var reason = task.reason
							if len(reason) == 0 {
								reason = "触发WAF规则自动加入"
							}

							pbItemMap[task.ip] = &pb.CreateIPItemsRequest_IPItem{
								IpListId:                      task.listId,
								IpFrom:                        task.ip,
								IpTo:                          "",
								ExpiredAt:                     task.expiresAt,
								Reason:                        reason,
								Type:                          ipType,
								EventLevel:                    task.level,
								ServerId:                      task.serverId,
								SourceNodeId:                  teaconst.NodeId,
								SourceServerId:                task.sourceServerId,
								SourceHTTPFirewallPolicyId:    task.sourceHTTPFirewallPolicyId,
								SourceHTTPFirewallRuleGroupId: task.sourceHTTPFirewallRuleGroupId,
								SourceHTTPFirewallRuleSetId:   task.sourceHTTPFirewallRuleSetId,
							}

							if len(pbItemMap) >= maxItems {
								return
							}
						default:
							return
						}
					}
				}()

				if len(pbItemMap) > 0 {
					var pbItems = []*pb.CreateIPItemsRequest_IPItem{}
					for _, pbItem := range pbItemMap {
						pbItems = append(pbItems, pbItem)
					}
					_, err = rpcClient.IPItemRPC.CreateIPItems(rpcClient.Context(), &pb.CreateIPItemsRequest{IpItems: pbItems})
					if err != nil {
						remotelogs.Error("WAF_RECORD_IP_ACTION", "create ip item failed: "+err.Error())
					}
				} else {
					time.Sleep(1 * time.Second)
				}
			}
		})
	})
}

type RecordIPAction struct {
	BaseAction

	Type            string `yaml:"type" json:"type"`
	IPListId        int64  `yaml:"ipListId" json:"ipListId"`
	IPListIsDeleted bool   `yaml:"ipListIsDeleted" json:"ipListIsDeleted"`
	Level           string `yaml:"level" json:"level"`
	Timeout         int32  `yaml:"timeout" json:"timeout"`
	Scope           string `yaml:"scope" json:"scope"`
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

func (this *RecordIPAction) Perform(waf *WAF, group *RuleGroup, set *RuleSet, request requests.Request, writer http.ResponseWriter) (continueRequest bool, goNextSet bool) {
	var ipListId = this.IPListId
	if ipListId <= 0 {
		ipListId = firewallconfigs.GlobalListId
	}

	// 是否已删除
	var ipListIsAvailable = (ipListId == firewallconfigs.GlobalListId) || (ipListId > 0 && !this.IPListIsDeleted && !ExistDeletedIPList(ipListId))

	// 是否在本地白名单中
	if SharedIPWhiteList.Contains("set:"+types.String(set.Id), this.Scope, request.WAFServerId(), request.WAFRemoteIP()) {
		return true, false
	}

	var timeout = this.Timeout
	var isForever = false
	if timeout <= 0 {
		isForever = true
		timeout = 86400 // 1天
	}
	var expiresAt = time.Now().Unix() + int64(timeout)

	if this.Type == "black" {
		request.ProcessResponseHeaders(writer.Header(), http.StatusForbidden)
		writer.WriteHeader(http.StatusForbidden)

		request.WAFClose()

		// 先加入本地的黑名单
		if ipListIsAvailable {
			SharedIPBlackList.Add(IPTypeAll, this.Scope, request.WAFServerId(), request.WAFRemoteIP(), expiresAt)
		}
	} else {
		// 加入本地白名单
		if ipListIsAvailable {
			SharedIPWhiteList.Add("set:"+types.String(set.Id), this.Scope, request.WAFServerId(), request.WAFRemoteIP(), expiresAt)
		}
	}

	// 上报
	if ipListId > 0 && ipListIsAvailable {
		var serverId int64
		if this.Scope == firewallconfigs.FirewallScopeService {
			serverId = request.WAFServerId()
		}

		var realExpiresAt = expiresAt
		if isForever {
			realExpiresAt = 0
		}

		select {
		case recordIPTaskChan <- &recordIPTask{
			ip:                            request.WAFRemoteIP(),
			listId:                        ipListId,
			expiresAt:                     realExpiresAt,
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

	return this.Type != "black", false
}

// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package agents

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/iwind/TeaGo/Tea"
	"net"
)

func init() {
	if !teaconst.IsMain {
		return
	}

	events.On(events.EventLoaded, func() {
		goman.New(func() {
			SharedQueue.Start()
		})
	})
}

var SharedQueue = NewQueue()

type Queue struct {
	c        chan string // chan ip
	cacheMap *IPCacheMap
}

func NewQueue() *Queue {
	return &Queue{
		c:        make(chan string, 128),
		cacheMap: NewIPCacheMap(65535),
	}
}

func (this *Queue) Start() {
	for ip := range this.c {
		err := this.Process(ip)
		if err != nil {
			// 不需要上报错误
			if Tea.IsTesting() {
				remotelogs.Debug("SharedParseQueue", err.Error())
			}
			continue
		}
	}
}

// Push 将IP加入到处理队列
func (this *Queue) Push(ip string) {
	// 是否在处理中
	if this.cacheMap.Contains(ip) {
		return
	}
	this.cacheMap.Add(ip)

	// 加入到队列
	select {
	case this.c <- ip:
	default:
	}
}

// Process 处理IP
func (this *Queue) Process(ip string) error {
	// 是否已经在库中
	if SharedManager.ContainsIP(ip) {
		return nil
	}

	ptr, err := this.ParseIP(ip)
	if err != nil {
		return err
	}
	if len(ptr) == 0 || ptr == "." {
		return nil
	}

	//remotelogs.Debug("AGENT", ip+" => "+ptr)

	var agentCode = this.ParsePtr(ptr)
	if len(agentCode) == 0 {
		return nil
	}

	// 加入到本地
	SharedManager.AddIP(ip, agentCode)

	var pbAgentIP = &pb.CreateClientAgentIPsRequest_AgentIPInfo{
		AgentCode: agentCode,
		Ip:        ip,
		Ptr:       ptr,
	}
	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return err
	}
	_, err = rpcClient.ClientAgentIPRPC.CreateClientAgentIPs(rpcClient.Context(), &pb.CreateClientAgentIPsRequest{AgentIPs: []*pb.CreateClientAgentIPsRequest_AgentIPInfo{pbAgentIP}})
	if err != nil {
		return err
	}

	return nil
}

// ParseIP 分析IP的PTR值
func (this *Queue) ParseIP(ip string) (ptr string, err error) {
	if len(ip) == 0 {
		return "", nil
	}

	names, err := net.LookupAddr(ip)
	if err != nil {
		return "", err
	}

	if len(names) == 0 {
		return "", nil
	}

	return names[0], nil
}

// ParsePtr 分析PTR对应的Agent
func (this *Queue) ParsePtr(ptr string) (agentCode string) {
	for _, agent := range AllAgents {
		if agent.Match(ptr) {
			return agent.Code
		}
	}
	return ""
}

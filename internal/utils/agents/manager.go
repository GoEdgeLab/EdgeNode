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
	"sync"
	"time"
)

var SharedManager = NewManager()

func init() {
	if !teaconst.IsMain {
		return
	}

	events.On(events.EventLoaded, func() {
		goman.New(func() {
			SharedManager.Start()
		})
	})
}

// Manager Agent管理器
type Manager struct {
	ipMap  map[string]string // ip => agentCode
	locker sync.RWMutex

	db *DB

	lastId int64
}

func NewManager() *Manager {
	return &Manager{
		ipMap: map[string]string{},
	}
}

func (this *Manager) SetDB(db *DB) {
	this.db = db
}

func (this *Manager) Start() {
	remotelogs.Println("AGENT_MANAGER", "starting ...")

	err := this.loadDB()
	if err != nil {
		remotelogs.Error("AGENT_MANAGER", "load database failed: "+err.Error())
		return
	}

	// 从本地数据库中加载
	err = this.Load()
	if err != nil {
		remotelogs.Error("AGENT_MANAGER", "load failed: "+err.Error())
	}

	// 先从API获取
	err = this.LoopAll()
	if err != nil {
		if rpc.IsConnError(err) {
			remotelogs.Debug("AGENT_MANAGER", "retrieve latest agent ip failed: "+err.Error())
		} else {
			remotelogs.Error("AGENT_MANAGER", "retrieve latest agent ip failed: "+err.Error())
		}
	}

	// 定时获取
	var duration = 30 * time.Minute
	if Tea.IsTesting() {
		duration = 30 * time.Second
	}
	var ticker = time.NewTicker(duration)
	for range ticker.C {
		err = this.LoopAll()
		if err != nil {
			remotelogs.Error("AGENT_MANAGER", "retrieve latest agent ip failed: "+err.Error())
		}
	}
}

func (this *Manager) Load() error {
	var offset int64 = 0
	var size int64 = 10000
	for {
		agentIPs, err := this.db.ListAgentIPs(offset, size)
		if err != nil {
			return err
		}
		if len(agentIPs) == 0 {
			break
		}
		for _, agentIP := range agentIPs {
			this.locker.Lock()
			this.ipMap[agentIP.IP] = agentIP.AgentCode
			this.locker.Unlock()

			if agentIP.Id > this.lastId {
				this.lastId = agentIP.Id
			}
		}
		offset += size
	}

	return nil
}

func (this *Manager) LoopAll() error {
	for {
		hasNext, err := this.Loop()
		if err != nil {
			return err
		}
		if !hasNext {
			break
		}
	}
	return nil
}

// Loop 单次循环获取数据
func (this *Manager) Loop() (hasNext bool, err error) {
	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return false, err
	}
	ipsResp, err := rpcClient.ClientAgentIPRPC.ListClientAgentIPsAfterId(rpcClient.Context(), &pb.ListClientAgentIPsAfterIdRequest{
		Id:   this.lastId,
		Size: 10000,
	})
	if err != nil {
		return false, err
	}
	if len(ipsResp.ClientAgentIPs) == 0 {
		return false, nil
	}
	for _, agentIP := range ipsResp.ClientAgentIPs {
		if agentIP.ClientAgent == nil {
			// 设置ID
			if agentIP.Id > this.lastId {
				this.lastId = agentIP.Id
			}

			continue
		}

		// 写入到数据库
		err = this.db.InsertAgentIP(agentIP.Id, agentIP.Ip, agentIP.ClientAgent.Code)
		if err != nil {
			return false, err
		}

		// 写入Map
		this.locker.Lock()
		this.ipMap[agentIP.Ip] = agentIP.ClientAgent.Code
		this.locker.Unlock()

		// 设置ID
		if agentIP.Id > this.lastId {
			this.lastId = agentIP.Id
		}
	}

	return true, nil
}

// AddIP 添加记录
func (this *Manager) AddIP(ip string, agentCode string) {
	this.locker.Lock()
	this.ipMap[ip] = agentCode
	this.locker.Unlock()
}

// LookupIP 查询IP所属Agent
func (this *Manager) LookupIP(ip string) (agentCode string) {
	this.locker.RLock()
	defer this.locker.RUnlock()
	return this.ipMap[ip]
}

// ContainsIP 检查是否有IP相关数据
func (this *Manager) ContainsIP(ip string) bool {
	this.locker.RLock()
	defer this.locker.RUnlock()
	_, ok := this.ipMap[ip]
	return ok
}

func (this *Manager) loadDB() error {
	var db = NewDB(Tea.Root + "/data/agents.db")
	err := db.Init()
	if err != nil {
		return err
	}
	this.db = db
	return nil
}

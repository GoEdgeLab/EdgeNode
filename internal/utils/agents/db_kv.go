// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package agents

import (
	"errors"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/utils/kvstore"
)

type KVDB struct {
	table   *kvstore.Table[*AgentIP]
	encoder *AgentIPEncoder[*AgentIP]
	lastKey string
}

func NewKVDB() *KVDB {
	var db = &KVDB{}

	events.OnClose(func() {
		_ = db.Close()
	})

	return db
}

func (this *KVDB) Init() error {
	store, err := kvstore.DefaultStore()
	if err != nil {
		return err
	}

	db, err := store.NewDB("agents")
	if err != nil {
		return err
	}

	{
		this.encoder = &AgentIPEncoder[*AgentIP]{}
		table, tableErr := kvstore.NewTable[*AgentIP]("agent_ips", this.encoder)
		if tableErr != nil {
			return tableErr
		}
		db.AddTable(table)
		this.table = table
	}

	return nil
}

func (this *KVDB) InsertAgentIP(ipId int64, ip string, agentCode string) error {
	if this.table == nil {
		return errors.New("table should not be nil")
	}

	var item = &AgentIP{
		Id:        ipId,
		IP:        ip,
		AgentCode: agentCode,
	}
	var key = this.encoder.EncodeKey(item)
	return this.table.Set(key, item)
}

func (this *KVDB) ListAgentIPs(offset int64, size int64) (agentIPs []*AgentIP, err error) {
	if this.table == nil {
		return nil, errors.New("table should not be nil")
	}

	err = this.table.
		Query().
		Limit(int(size)).
		Offset(this.lastKey).
		FindAll(func(tx *kvstore.Tx[*AgentIP], item kvstore.Item[*AgentIP]) (goNext bool, err error) {
			this.lastKey = item.Key
			agentIPs = append(agentIPs, item.Value)
			return true, nil
		})

	return
}

func (this *KVDB) Close() error {
	return nil
}

func (this *KVDB) Flush() error {
	if this.table == nil {
		return errors.New("table should not be nil")
	}

	return this.table.DB().Store().Flush()
}

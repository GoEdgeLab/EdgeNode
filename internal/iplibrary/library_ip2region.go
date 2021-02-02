package iplibrary

import (
	"fmt"
	"github.com/TeaOSLab/EdgeNode/internal/errors"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/lionsoul2014/ip2region/binding/golang/ip2region"
	"strings"
)

type IP2RegionLibrary struct {
	db *ip2region.Ip2Region
}

func (this *IP2RegionLibrary) Load(dbPath string) error {
	db, err := ip2region.New(dbPath)
	if err != nil {
		return err
	}
	this.db = db

	return nil
}

func (this *IP2RegionLibrary) Lookup(ip string) (*Result, error) {
	// 暂不支持IPv6
	if strings.Contains(ip, ":") {
		return nil, nil
	}

	if this.db == nil {
		return nil, errors.New("library has not been loaded")
	}

	defer func() {
		// 防止panic发生
		err := recover()
		if err != nil {
			remotelogs.Error("IP2RegionLibrary", "panic: "+fmt.Sprintf("%#v", err))
		}
	}()

	info, err := this.db.MemorySearch(ip)
	if err != nil {
		return nil, err
	}

	if info.Country == "0" {
		info.Country = ""
	}
	if info.Region == "0" {
		info.Region = ""
	}
	if info.Province == "0" {
		info.Province = ""
	}
	if info.City == "0" {
		info.City = ""
	}
	if info.ISP == "0" {
		info.ISP = ""
	}

	return &Result{
		CityId:   info.CityId,
		Country:  info.Country,
		Region:   info.Region,
		Province: info.Province,
		City:     info.City,
		ISP:      info.ISP,
	}, nil
}

func (this *IP2RegionLibrary) Close() {
	if this.db != nil {
		this.db.Close()
	}
}

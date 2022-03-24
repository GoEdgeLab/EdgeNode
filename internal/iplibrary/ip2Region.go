// 源码改自：https://github.com/lionsoul2014/ip2region/blob/master/binding/golang/ip2region/ip2Region.go

package iplibrary

import (
	"errors"
	"io/ioutil"
	"strconv"
	"strings"
)

const (
	IndexBlockLength = 12
)

var err error

type IP2Region struct {
	headerSip []int64
	headerPtr []int64
	headerLen int64

	// super block index info
	firstIndexPtr int64
	lastIndexPtr  int64
	totalBlocks   int64

	dbData []byte
}

type IpInfo struct {
	CityId   int64
	Country  string
	Region   string
	Province string
	City     string
	ISP      string
}

func (ip IpInfo) String() string {
	return strconv.FormatInt(ip.CityId, 10) + "|" + ip.Country + "|" + ip.Region + "|" + ip.Province + "|" + ip.City + "|" + ip.ISP
}

func getIpInfo(cityId int64, line []byte) *IpInfo {
	lineSlice := strings.Split(string(line), "|")
	ipInfo := &IpInfo{}
	length := len(lineSlice)
	ipInfo.CityId = cityId
	if length < 5 {
		for i := 0; i <= 5-length; i++ {
			lineSlice = append(lineSlice, "")
		}
	}

	ipInfo.Country = lineSlice[0]
	ipInfo.Region = lineSlice[1]
	ipInfo.Province = lineSlice[2]
	ipInfo.City = lineSlice[3]
	ipInfo.ISP = lineSlice[4]
	return ipInfo
}

func NewIP2Region(path string) (*IP2Region, error) {
	var region = &IP2Region{}
	region.dbData, err = ioutil.ReadFile(path)

	if err != nil {
		return nil, err
	}

	region.firstIndexPtr = region.ipLongAtOffset(0)
	region.lastIndexPtr = region.ipLongAtOffset(4)
	region.totalBlocks = (region.lastIndexPtr-region.firstIndexPtr)/IndexBlockLength + 1
	return region, nil
}

func (this *IP2Region) MemorySearch(ipStr string) (ipInfo *IpInfo, err error) {
	ip, err := ip2long(ipStr)
	if err != nil {
		return nil, err
	}

	h := this.totalBlocks
	var dataPtr, l int64
	for l <= h {
		m := (l + h) >> 1
		p := this.firstIndexPtr + m*IndexBlockLength
		sip := this.ipLongAtOffset(p)
		if ip < sip {
			h = m - 1
		} else {
			eip := this.ipLongAtOffset(p + 4)
			if ip > eip {
				l = m + 1
			} else {
				dataPtr = this.ipLongAtOffset(p + 8)
				break
			}
		}
	}
	if dataPtr == 0 {
		return nil, nil
	}

	dataLen := (dataPtr >> 24) & 0xFF
	dataPtr = dataPtr & 0x00FFFFFF
	return getIpInfo(this.ipLongAtOffset(dataPtr), this.dbData[(dataPtr)+4:dataPtr+dataLen]), nil
}

func (this *IP2Region) ipLongAtOffset(offset int64) int64 {
	return int64(this.dbData[offset]) |
		int64(this.dbData[offset+1])<<8 |
		int64(this.dbData[offset+2])<<16 |
		int64(this.dbData[offset+3])<<24
}

func ip2long(IpStr string) (int64, error) {
	bits := strings.Split(IpStr, ".")
	if len(bits) != 4 {
		return 0, errors.New("ip format error")
	}

	var sum int64
	for i, n := range bits {
		bit, _ := strconv.ParseInt(n, 10, 64)
		sum += bit << uint(24-8*i)
	}

	return sum, nil
}

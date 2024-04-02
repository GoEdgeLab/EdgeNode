// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package metrics

import (
	"bytes"
	"encoding/binary"
	"errors"
	byteutils "github.com/TeaOSLab/EdgeNode/internal/utils/byte"
	"github.com/TeaOSLab/EdgeNode/internal/utils/fnv"
	"strconv"
	"strings"
)

type Stat struct {
	ServerId int64    `json:"serverId"`
	Keys     []string `json:"keys"`
	Hash     string   `json:"hash"`
	Value    int64    `json:"value"`
	Time     string   `json:"time"`
}

func UniqueKey(serverId int64, keys []string, time string, version int32, itemId int64) string {
	var keysData = strings.Join(keys, "$EDGE$")
	return strconv.FormatUint(fnv.HashString(strconv.FormatInt(serverId, 10)+"@"+keysData+"@"+time+"@"+strconv.Itoa(int(version))+"@"+strconv.FormatInt(itemId, 10)), 10)
}

func (this *Stat) UniqueKey(version int32, itemId int64) string {
	return UniqueKey(this.ServerId, this.Keys, this.Time, version, itemId)
}

func (this *Stat) FullKey(version int32, itemId int64) string {
	return this.Time + "_" + string(int32ToBigEndian(version)) + this.UniqueKey(version, itemId)
}

func (this *Stat) EncodeValueKey(version int32) string {
	if this.Value < 0 {
		this.Value = 0
	}

	return string(byteutils.Concat([]byte(this.Time), []byte{'_'}, int32ToBigEndian(version), int64ToBigEndian(this.ServerId), int64ToBigEndian(this.Value), []byte(this.Hash)))
}

func (this *Stat) EncodeSumKey(version int32) string {
	return string(byteutils.Concat([]byte(this.Time), []byte{'_'}, int32ToBigEndian(version), int64ToBigEndian(this.ServerId)))
}

func DecodeValueKey(valueKey string) (serverId int64, timeString string, version int32, value int64, hash string, err error) {
	var b = []byte(valueKey)
	var timeIndex = bytes.Index(b, []byte{'_'})
	if timeIndex < 0 {
		return
	}

	timeString = string(b[:timeIndex])
	b = b[timeIndex+1:]

	if len(b) < 20+1 {
		err = errors.New("invalid value key")
		return
	}

	version = int32(binary.BigEndian.Uint32(b[0:4]))
	serverId = int64(binary.BigEndian.Uint64(b[4:12]))
	value = int64(binary.BigEndian.Uint64(b[12:20]))
	hash = string(b[20:])
	return
}

func DecodeSumKey(sumKey string) (serverId int64, timeString string, version int32, err error) {
	var b = []byte(sumKey)
	var timeIndex = bytes.Index(b, []byte{'_'})
	if timeIndex < 0 {
		return
	}

	timeString = string(b[:timeIndex])
	b = b[timeIndex+1:]

	if len(b) < 12 {
		err = errors.New("invalid sum key")
		return
	}

	version = int32(binary.BigEndian.Uint32(b[:4]))
	serverId = int64(binary.BigEndian.Uint64(b[4:12]))
	return
}

func EncodeSumValue(count uint64, total uint64) []byte {
	var result [16]byte
	binary.BigEndian.PutUint64(result[:8], count)
	binary.BigEndian.PutUint64(result[8:], total)
	return result[:]
}

func DecodeSumValue(data []byte) (count uint64, total uint64) {
	if len(data) != 16 {
		return
	}
	count = binary.BigEndian.Uint64(data[:8])
	total = binary.BigEndian.Uint64(data[8:])
	return
}

func int64ToBigEndian(i int64) []byte {
	if i < 0 {
		i = 0
	}
	var b = make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(i))
	return b
}

func int32ToBigEndian(i int32) []byte {
	if i < 0 {
		i = 0
	}
	var b = make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(i))
	return b
}

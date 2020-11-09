package iplibrary

import (
	"fmt"
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/errors"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/logs"
	"os"
	"time"
)

func init() {
	events.On(events.EventStart, func() {
		updater := NewUpdater()
		updater.Start()
	})
}

// IP库更新程序
type Updater struct {
}

// 获取新对象
func NewUpdater() *Updater {
	return &Updater{}
}

// 开始更新
func (this *Updater) Start() {
	// 这里不需要太频繁检查更新，因为通常不需要更新IP库
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			err := this.loop()
			if err != nil {
				logs.Println("[IP_LIBRARY]" + err.Error())
			}
		}
	}()
}

// 单次任务
func (this *Updater) loop() error {
	nodeConfig, err := nodeconfigs.SharedNodeConfig()
	if err != nil {
		return err
	}
	if nodeConfig.GlobalConfig == nil {
		return nil
	}
	code := nodeConfig.GlobalConfig.IPLibrary.Code
	if len(code) == 0 {
		code = serverconfigs.DefaultIPLibraryType
	}

	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return err
	}
	libraryResp, err := rpcClient.IPLibraryRPC().FindLatestIPLibraryWithType(rpcClient.Context(), &pb.FindLatestIPLibraryWithTypeRequest{Type: code})
	if err != nil {
		return err
	}
	lib := libraryResp.IpLibrary
	if lib == nil || lib.File == nil {
		return nil
	}

	typeInfo := serverconfigs.FindIPLibraryWithType(code)
	if typeInfo == nil {
		return errors.New("invalid ip library code '" + code + "'")
	}

	path := Tea.Root + "/resources/ipdata/" + code + "/" + code + "." + fmt.Sprintf("%d", lib.CreatedAt) + typeInfo.GetString("ext")

	// 是否已经存在
	_, err = os.Stat(path)
	if err == nil {
		return nil
	}

	// 开始下载
	fileChunkIdsResp, err := rpcClient.FileChunkRPC().FindAllFileChunkIds(rpcClient.Context(), &pb.FindAllFileChunkIdsRequest{FileId: lib.File.Id})
	if err != nil {
		return err
	}
	chunkIds := fileChunkIdsResp.FileChunkIds
	if len(chunkIds) == 0 {
		return nil
	}
	isOk := false

	fp, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}

	defer func() {
		// 如果保存不成功就直接删除
		if !isOk {
			_ = fp.Close()
			_ = os.Remove(path)
		}
	}()
	for _, chunkId := range chunkIds {
		chunkResp, err := rpcClient.FileChunkRPC().DownloadFileChunk(rpcClient.Context(), &pb.DownloadFileChunkRequest{FileChunkId: chunkId})
		if err != nil {
			return err
		}
		chunk := chunkResp.FileChunk

		if chunk == nil {
			continue
		}
		_, err = fp.Write(chunk.Data)
		if err != nil {
			return err
		}
	}

	err = fp.Close()
	if err != nil {
		return err
	}

	// 重新加载
	library, err := SharedManager.Load()
	if err != nil {
		return err
	}
	SharedLibrary = library

	isOk = true

	return nil
}

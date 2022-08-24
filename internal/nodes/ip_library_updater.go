// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved. Official site: https://goedge.cn .

package nodes

import (
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/types"
	"io"
	"os"
)

type IPLibraryUpdater struct {
}

func NewIPLibraryUpdater() *IPLibraryUpdater {
	return &IPLibraryUpdater{}
}

// DataDir 文件目录
func (this *IPLibraryUpdater) DataDir() string {
	// data/
	var dir = Tea.Root + "/data"
	stat, err := os.Stat(dir)
	if err == nil && stat.IsDir() {
		return dir
	}

	err = os.Mkdir(dir, 0666)
	if err == nil {
		return dir
	}

	remotelogs.Error("IP_LIBRARY_UPDATER", "create directory '"+dir+"' failed: "+err.Error())

	// 如果不能创建 data/ 目录，那么使用临时目录
	return os.TempDir()
}

// FindLatestFile 检查最新的IP库文件
func (this *IPLibraryUpdater) FindLatestFile() (code string, fileId int64, err error) {
	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return "", 0, err
	}
	resp, err := rpcClient.IPLibraryArtifactRPC.FindPublicIPLibraryArtifact(rpcClient.Context(), &pb.FindPublicIPLibraryArtifactRequest{})
	if err != nil {
		return "", 0, err
	}
	var artifact = resp.IpLibraryArtifact
	if artifact == nil {
		return
	}
	return artifact.Code, artifact.FileId, nil
}

// DownloadFile 下载文件
func (this *IPLibraryUpdater) DownloadFile(fileId int64, writer io.Writer) error {
	if fileId <= 0 {
		return errors.New("invalid fileId: " + types.String(fileId))
	}

	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return err
	}

	chunkIdsResp, err := rpcClient.FileChunkRPC.FindAllFileChunkIds(rpcClient.Context(), &pb.FindAllFileChunkIdsRequest{FileId: fileId})
	if err != nil {
		return err
	}
	for _, chunkId := range chunkIdsResp.FileChunkIds {
		chunkResp, err := rpcClient.FileChunkRPC.DownloadFileChunk(rpcClient.Context(), &pb.DownloadFileChunkRequest{FileChunkId: chunkId})
		if err != nil {
			return err
		}
		var chunk = chunkResp.FileChunk
		if chunk == nil {
			return errors.New("can not find file chunk with chunk id '" + types.String(chunkId) + "'")
		}
		_, err = writer.Write(chunk.Data)
		if err != nil {
			return err
		}
	}
	return nil
}

// LogInfo 普通日志
func (this *IPLibraryUpdater) LogInfo(message string) {
	remotelogs.Println("IP_LIBRARY_UPDATER", message)
}

// LogError 错误日志
func (this *IPLibraryUpdater) LogError(err error) {
	if err == nil {
		return
	}
	remotelogs.Error("IP_LIBRARY_UPDATER", err.Error())
}

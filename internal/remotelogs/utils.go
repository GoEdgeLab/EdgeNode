package remotelogs

import (
	"encoding/json"
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/goman"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"github.com/TeaOSLab/EdgeNode/internal/trackers"
	"github.com/cespare/xxhash"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
	"strings"
	"time"
)

var logChan = make(chan *pb.NodeLog, 1024)

func init() {
	// 定期上传日志
	ticker := time.NewTicker(60 * time.Second)
	if Tea.IsTesting() {
		ticker = time.NewTicker(10 * time.Second)
	}
	goman.New(func() {
		for range ticker.C {
			var tr = trackers.Begin("UPLOAD_REMOTE_LOGS")
			err := uploadLogs()
			tr.End()
			if err != nil {
				logs.Println("[LOG]" + err.Error())
			}
		}
	})
}

// Println 打印普通信息
func Println(tag string, description string) {
	logs.Println("[" + tag + "]" + description)

	select {
	case logChan <- &pb.NodeLog{
		Role:        teaconst.Role,
		Tag:         tag,
		Description: description,
		Level:       "info",
		NodeId:      teaconst.NodeId,
		CreatedAt:   time.Now().Unix(),
	}:
	default:

	}
}

// Warn 打印警告信息
func Warn(tag string, description string) {
	logs.Println("[" + tag + "]" + description)

	select {
	case logChan <- &pb.NodeLog{
		Role:        teaconst.Role,
		Tag:         tag,
		Description: description,
		Level:       "warning",
		NodeId:      teaconst.NodeId,
		CreatedAt:   time.Now().Unix(),
	}:
	default:

	}
}

// Error 打印错误信息
func Error(tag string, description string) {
	logs.Println("[" + tag + "]" + description)

	// 忽略RPC连接错误
	var level = "error"
	if strings.Contains(description, "code = Unavailable desc") {
		level = "warning"
	}

	select {
	case logChan <- &pb.NodeLog{
		Role:        teaconst.Role,
		Tag:         tag,
		Description: description,
		Level:       level,
		NodeId:      teaconst.NodeId,
		CreatedAt:   time.Now().Unix(),
	}:
	default:

	}
}

// ErrorObject 打印错误对象
func ErrorObject(tag string, err error) {
	if err == nil {
		return
	}
	if rpc.IsConnError(err) {
		Warn(tag, err.Error())
	} else {
		Error(tag, err.Error())
	}
}

// ServerError 打印服务相关错误信息
func ServerError(serverId int64, tag string, description string, logType nodeconfigs.NodeLogType, params maps.Map) {
	logs.Println("[" + tag + "]" + description)

	// 参数
	var paramsJSON []byte
	if len(params) > 0 {
		p, err := json.Marshal(params)
		if err != nil {
			logs.Println("[LOG]" + err.Error())
		} else {
			paramsJSON = p
		}
	}

	select {
	case logChan <- &pb.NodeLog{
		Role:        teaconst.Role,
		Tag:         tag,
		Description: description,
		Level:       "error",
		NodeId:      teaconst.NodeId,
		ServerId:    serverId,
		CreatedAt:   time.Now().Unix(),
		Type:        logType,
		ParamsJSON:  paramsJSON,
	}:
	default:

	}
}

// ServerSuccess 打印服务相关成功信息
func ServerSuccess(serverId int64, tag string, description string, logType nodeconfigs.NodeLogType, params maps.Map) {
	logs.Println("[" + tag + "]" + description)

	// 参数
	var paramsJSON []byte
	if len(params) > 0 {
		p, err := json.Marshal(params)
		if err != nil {
			logs.Println("[LOG]" + err.Error())
		} else {
			paramsJSON = p
		}
	}

	select {
	case logChan <- &pb.NodeLog{
		Role:        teaconst.Role,
		Tag:         tag,
		Description: description,
		Level:       "success",
		NodeId:      teaconst.NodeId,
		ServerId:    serverId,
		CreatedAt:   time.Now().Unix(),
		Type:        logType,
		ParamsJSON:  paramsJSON,
	}:
	default:

	}
}

// ServerLog 打印服务相关日志信息
func ServerLog(serverId int64, tag string, description string, logType nodeconfigs.NodeLogType, params maps.Map) {
	logs.Println("[" + tag + "]" + description)

	// 参数
	var paramsJSON []byte
	if len(params) > 0 {
		p, err := json.Marshal(params)
		if err != nil {
			logs.Println("[LOG]" + err.Error())
		} else {
			paramsJSON = p
		}
	}

	select {
	case logChan <- &pb.NodeLog{
		Role:        teaconst.Role,
		Tag:         tag,
		Description: description,
		Level:       "info",
		NodeId:      teaconst.NodeId,
		ServerId:    serverId,
		CreatedAt:   time.Now().Unix(),
		Type:        logType,
		ParamsJSON:  paramsJSON,
	}:
	default:

	}
}

// 上传日志
func uploadLogs() error {
	logList := []*pb.NodeLog{}

	const hashSize = 5
	var hashList = []uint64{}

Loop:
	for {
		select {
		case log := <-logChan:
			// 是否已存在
			var hash = xxhash.Sum64String(types.String(log.ServerId) + "_" + log.Description)
			var found = false
			for _, h := range hashList {
				if h == hash {
					found = true
					break
				}
			}

			// 加入
			if !found {
				hashList = append(hashList, hash)
				if len(hashList) > hashSize {
					hashList = hashList[1:]
				}

				logList = append(logList, log)
			}
		default:
			break Loop
		}
	}
	if len(logList) == 0 {
		return nil
	}
	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		return err
	}

	// 正在退出时不上报错误
	if teaconst.IsQuiting {
		return nil
	}

	_, err = rpcClient.NodeLogRPC.CreateNodeLogs(rpcClient.Context(), &pb.CreateNodeLogsRequest{NodeLogs: logList})
	return err
}

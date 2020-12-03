package iplibrary

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/errors"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/files"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/types"
	"regexp"
	"strings"
)

var SharedManager = NewManager()
var SharedLibrary LibraryInterface

func init() {
	events.On(events.EventLoaded, func() {
		// 初始化
		library, err := SharedManager.Load()
		if err != nil {
			logs.Println("[IP_LIBRARY]" + err.Error())
			return
		}
		SharedLibrary = library
	})
}

type Manager struct {
	code string
}

func NewManager() *Manager {
	return &Manager{}
}

func (this *Manager) Load() (LibraryInterface, error) {
	nodeConfig, err := nodeconfigs.SharedNodeConfig()
	if err != nil {
		return nil, err
	}
	config := nodeConfig.GlobalConfig
	if config == nil {
		config = &serverconfigs.GlobalConfig{}
	}

	// 当前正在使用的IP库代号
	code := config.IPLibrary.Code
	if len(code) == 0 {
		code = serverconfigs.DefaultIPLibraryType
	}

	dir := Tea.Root + "/resources/ipdata/" + code
	var lastVersion int64 = -1
	lastFilename := ""
	for _, file := range files.NewFile(dir).List() {
		filename := file.Name()

		reg := regexp.MustCompile(`^` + regexp.QuoteMeta(code) + `.(\d+)\.`)
		if reg.MatchString(filename) { // 先查找有版本号的
			result := reg.FindStringSubmatch(filename)
			version := types.Int64(result[1])
			if version > lastVersion {
				lastVersion = version
				lastFilename = filename
			}
		} else if strings.HasPrefix(filename, code+".") { // 后查找默认的
			if lastVersion == -1 {
				lastFilename = filename
				lastVersion = 0
			}
		}
	}

	if len(lastFilename) == 0 {
		return nil, errors.New("ip library file not found")
	}

	var libraryPtr LibraryInterface
	switch code {
	case serverconfigs.IPLibraryTypeIP2Region:
		libraryPtr = &IP2RegionLibrary{}
	default:
		return nil, errors.New("invalid ip library code '" + code + "'")
	}

	err = libraryPtr.Load(dir + "/" + lastFilename)
	if err != nil {
		return nil, err
	}

	return libraryPtr, nil
}

package serverconfigs

import (
	"github.com/TeaOSLab/EdgeNode/internal/configs/serverconfigs/configutils"
	"github.com/TeaOSLab/EdgeNode/internal/configs/serverconfigs/shared"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/files"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/logs"
	"strconv"
	"strings"
	"time"
)

var DefaultSkippedResponseCacheControlValues = []string{"private", "no-cache", "no-store"}

// 缓存策略配置
type CachePolicy struct {
	Id   int    `yaml:"id" json:"id"`
	IsOn bool   `yaml:"isOn" json:"isOn"` // 是否开启 TODO
	Name string `yaml:"name" json:"name"` // 名称

	Key      string              `yaml:"key" json:"key"`           // 每个缓存的Key规则，里面可以有变量
	Capacity shared.SizeCapacity `yaml:"capacity" json:"capacity"` // 最大内容容量
	Life     shared.TimeDuration `yaml:"life" json:"life"`         // 时间
	Status   []int               `yaml:"status" json:"status"`     // 缓存的状态码列表
	MaxSize  shared.SizeCapacity `yaml:"maxSize" json:"maxSize"`   // 能够请求的最大尺寸

	SkipResponseCacheControlValues []string `yaml:"skipCacheControlValues" json:"skipCacheControlValues"`     // 可以跳过的响应的Cache-Control值
	SkipResponseSetCookie          bool     `yaml:"skipSetCookie" json:"skipSetCookie"`                       // 是否跳过响应的Set-Cookie Header
	EnableRequestCachePragma       bool     `yaml:"enableRequestCachePragma" json:"enableRequestCachePragma"` // 是否支持客户端的Pragma: no-cache

	Cond []*shared.RequestCond `yaml:"cond" json:"cond"`

	life     time.Duration
	maxSize  int64
	capacity int64

	uppercaseSkipCacheControlValues []string

	Type    string                 `yaml:"type" json:"type"`       // 类型
	Options map[string]interface{} `yaml:"options" json:"options"` // 选项
}

// 获取新对象
func NewCachePolicy() *CachePolicy {
	return &CachePolicy{
		SkipResponseCacheControlValues: DefaultSkippedResponseCacheControlValues,
		SkipResponseSetCookie:          true,
	}
}

// 从文件中读取缓存策略
func NewCachePolicyFromFile(file string) *CachePolicy {
	if len(file) == 0 {
		return nil
	}
	reader, err := files.NewReader(Tea.ConfigFile(file))
	if err != nil {
		logs.Error(err)
		return nil
	}
	defer func() {
		_ = reader.Close()
	}()

	p := NewCachePolicy()
	err = reader.ReadYAML(p)
	if err != nil {
		logs.Error(err)
		return nil
	}

	return p
}

// 校验
func (this *CachePolicy) Validate() error {
	var err error
	this.maxSize = this.MaxSize.Bytes()
	this.life = this.Life.Duration()
	this.capacity = this.Capacity.Bytes()

	this.uppercaseSkipCacheControlValues = []string{}
	for _, value := range this.SkipResponseCacheControlValues {
		this.uppercaseSkipCacheControlValues = append(this.uppercaseSkipCacheControlValues, strings.ToUpper(value))
	}

	// cond
	if len(this.Cond) > 0 {
		for _, cond := range this.Cond {
			err := cond.Validate()
			if err != nil {
				return err
			}
		}
	}

	return err
}

// 最大数据尺寸
func (this *CachePolicy) MaxDataSize() int64 {
	return this.maxSize
}

// 容量
func (this *CachePolicy) CapacitySize() int64 {
	return this.capacity
}

// 生命周期
func (this *CachePolicy) LifeDuration() time.Duration {
	return this.life
}

// 保存
func (this *CachePolicy) Save() error {
	shared.Locker.Lock()
	defer shared.Locker.Unlock()

	filename := "cache.policy." + strconv.Itoa(this.Id) + ".conf"
	writer, err := files.NewWriter(Tea.ConfigFile(filename))
	if err != nil {
		return err
	}
	defer func() {
		_ = writer.Close()
	}()
	_, err = writer.WriteYAML(this)
	return err
}

// 删除
func (this *CachePolicy) Delete() error {
	filename := "cache.policy." + strconv.Itoa(this.Id) + ".conf"
	return files.NewFile(Tea.ConfigFile(filename)).Delete()
}

// 是否包含某个Cache-Control值
func (this *CachePolicy) ContainsCacheControl(value string) bool {
	return lists.ContainsString(this.uppercaseSkipCacheControlValues, strings.ToUpper(value))
}

// 检查是否匹配关键词
func (this *CachePolicy) MatchKeyword(keyword string) (matched bool, name string, tags []string) {
	if configutils.MatchKeyword(this.Name, keyword) || configutils.MatchKeyword(this.Type, keyword) {
		matched = true
		name = this.Name
		if len(this.Type) > 0 {
			tags = []string{"类型：" + this.Type}
		}
	}
	return
}

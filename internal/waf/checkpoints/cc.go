package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/utils/counters"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/TeaOSLab/EdgeNode/internal/waf/utils"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
	"regexp"
)

// CCCheckpoint ${cc.arg}
// TODO implement more traffic rules
type CCCheckpoint struct {
	Checkpoint
}

func (this *CCCheckpoint) Init() {

}

func (this *CCCheckpoint) Start() {

}

func (this *CCCheckpoint) RequestValue(req requests.Request, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	value = 0

	periodString, ok := options["period"]
	if !ok {
		return
	}
	var period = types.Int(periodString)
	if period < 1 {
		return
	}

	if period > 7*86400 {
		period = 7 * 86400
	}

	var userType = types.String(options["userType"])
	var userField = types.String(options["userField"])
	var userIndex = types.Int(options["userIndex"])

	if param == "requests" { // requests
		var key = ""
		switch userType {
		case "ip":
			key = req.WAFRemoteIP()
		case "cookie":
			if len(userField) == 0 {
				key = req.WAFRemoteIP()
			} else {
				cookie, _ := req.WAFRaw().Cookie(userField)
				if cookie != nil {
					v := cookie.Value
					if userIndex > 0 && len(v) > userIndex {
						v = v[userIndex:]
					}
					key = "USER@" + userType + "@" + userField + "@" + v
				}
			}
		case "get":
			if len(userField) == 0 {
				key = req.WAFRemoteIP()
			} else {
				v := req.WAFRaw().URL.Query().Get(userField)
				if userIndex > 0 && len(v) > userIndex {
					v = v[userIndex:]
				}
				key = "USER@" + userType + "@" + userField + "@" + v
			}
		case "post":
			if len(userField) == 0 {
				key = req.WAFRemoteIP()
			} else {
				v := req.WAFRaw().PostFormValue(userField)
				if userIndex > 0 && len(v) > userIndex {
					v = v[userIndex:]
				}
				key = "USER@" + userType + "@" + userField + "@" + v
			}
		case "header":
			if len(userField) == 0 {
				key = req.WAFRemoteIP()
			} else {
				v := req.WAFRaw().Header.Get(userField)
				if userIndex > 0 && len(v) > userIndex {
					v = v[userIndex:]
				}
				key = "USER@" + userType + "@" + userField + "@" + v
			}
		default:
			key = req.WAFRemoteIP()
		}
		if len(key) == 0 {
			key = req.WAFRemoteIP()
		}
		value = counters.SharedCounter.IncreaseKey(types.String(ruleId)+"@WAF_CC@"+key, types.Int(period))
	}

	return
}

func (this *CCCheckpoint) ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map, ruleId int64) (value any, hasRequestBody bool, sysErr error, userErr error) {
	if this.IsRequest() {
		return this.RequestValue(req, param, options, ruleId)
	}
	return
}

func (this *CCCheckpoint) ParamOptions() *ParamOptions {
	option := NewParamOptions()
	option.AddParam("请求数", "requests")
	return option
}

func (this *CCCheckpoint) Options() []OptionInterface {
	options := []OptionInterface{}

	// period
	{
		option := NewFieldOption("统计周期", "period")
		option.Value = "60"
		option.RightLabel = "秒"
		option.Size = 8
		option.MaxLength = 8
		option.Validate = func(value string) (ok bool, message string) {
			if regexp.MustCompile(`^\d+$`).MatchString(value) {
				ok = true
				return
			}
			message = "周期需要是一个整数数字"
			return
		}
		options = append(options, option)
	}

	// type
	{
		option := NewOptionsOption("用户识别读取来源", "userType")
		option.Size = 10
		option.SetOptions([]maps.Map{
			{
				"name":  "IP",
				"value": "ip",
			},
			{
				"name":  "Cookie",
				"value": "cookie",
			},
			{
				"name":  "URL参数",
				"value": "get",
			},
			{
				"name":  "POST参数",
				"value": "post",
			},
			{
				"name":  "HTTP Header",
				"value": "header",
			},
		})
		options = append(options, option)
	}

	// user field
	{
		option := NewFieldOption("用户识别字段", "userField")
		option.Comment = "识别用户的唯一性字段，在用户读取来源不是IP时使用"
		options = append(options, option)
	}

	// user value index
	{
		option := NewFieldOption("字段读取位置", "userIndex")
		option.Size = 5
		option.MaxLength = 5
		option.Comment = "读取用户识别字段的位置，从0开始，比如user12345的数字ID 12345的位置就是5，在用户读取来源不是IP时使用"
		options = append(options, option)
	}

	return options
}

func (this *CCCheckpoint) Stop() {

}

func (this *CCCheckpoint) CacheLife() utils.CacheLife {
	return utils.CacheDisabled
}

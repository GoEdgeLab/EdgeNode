package waf

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/configutils"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/filterconfigs"
	"github.com/TeaOSLab/EdgeNode/internal/re"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/waf/checkpoints"
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/TeaOSLab/EdgeNode/internal/waf/utils"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
	"github.com/iwind/TeaGo/utils/string"
	"net"
	"reflect"
	"regexp"
	"strings"
)

var singleParamRegexp = regexp.MustCompile("^\\${[\\w.-]+}$")

// Rule
type Rule struct {
	Id int64

	Description       string                 `yaml:"description" json:"description"`
	Param             string                 `yaml:"param" json:"param"` // such as ${arg.name} or ${args}, can be composite as ${arg.firstName}${arg.lastName}
	ParamFilters      []*ParamFilter         `yaml:"paramFilters" json:"paramFilters"`
	Operator          RuleOperator           `yaml:"operator" json:"operator"` // such as contains, gt,  ...
	Value             string                 `yaml:"value" json:"value"`       // compared value
	IsCaseInsensitive bool                   `yaml:"isCaseInsensitive" json:"isCaseInsensitive"`
	CheckpointOptions map[string]interface{} `yaml:"checkpointOptions" json:"checkpointOptions"`

	checkpointFinder func(prefix string) checkpoints.CheckpointInterface

	singleParam      string                          // real param after prefix
	singleCheckpoint checkpoints.CheckpointInterface // if is single check point

	multipleCheckpoints map[string]checkpoints.CheckpointInterface

	isIP    bool
	ipValue net.IP

	floatValue float64
	reg        *re.Regexp
}

func NewRule() *Rule {
	return &Rule{}
}

func (this *Rule) Init() error {
	// operator
	switch this.Operator {
	case RuleOperatorGt:
		this.floatValue = types.Float64(this.Value)
	case RuleOperatorGte:
		this.floatValue = types.Float64(this.Value)
	case RuleOperatorLt:
		this.floatValue = types.Float64(this.Value)
	case RuleOperatorLte:
		this.floatValue = types.Float64(this.Value)
	case RuleOperatorEq:
		this.floatValue = types.Float64(this.Value)
	case RuleOperatorNeq:
		this.floatValue = types.Float64(this.Value)
	case RuleOperatorMatch:
		v := this.Value
		if this.IsCaseInsensitive && !strings.HasPrefix(v, "(?i)") {
			v = "(?i)" + v
		}

		v = this.unescape(v)

		reg, err := re.Compile(v)
		if err != nil {
			return err
		}
		this.reg = reg
	case RuleOperatorNotMatch:
		v := this.Value
		if this.IsCaseInsensitive && !strings.HasPrefix(v, "(?i)") {
			v = "(?i)" + v
		}

		v = this.unescape(v)

		reg, err := re.Compile(v)
		if err != nil {
			return err
		}
		this.reg = reg
	case RuleOperatorEqIP, RuleOperatorGtIP, RuleOperatorGteIP, RuleOperatorLtIP, RuleOperatorLteIP:
		this.ipValue = net.ParseIP(this.Value)
		this.isIP = this.ipValue != nil

		if !this.isIP {
			return errors.New("value should be a valid ip")
		}
	case RuleOperatorIPRange, RuleOperatorNotIPRange:
		if strings.Contains(this.Value, ",") {
			ipList := strings.SplitN(this.Value, ",", 2)
			ipString1 := strings.TrimSpace(ipList[0])
			ipString2 := strings.TrimSpace(ipList[1])

			if len(ipString1) > 0 {
				ip1 := net.ParseIP(ipString1)
				if ip1 == nil {
					return errors.New("start ip is invalid")
				}
			}

			if len(ipString2) > 0 {
				ip2 := net.ParseIP(ipString2)
				if ip2 == nil {
					return errors.New("end ip is invalid")
				}
			}
		} else if strings.Contains(this.Value, "/") {
			_, _, err := net.ParseCIDR(this.Value)
			if err != nil {
				return err
			}
		} else {
			return errors.New("invalid ip range")
		}
	}

	if singleParamRegexp.MatchString(this.Param) {
		param := this.Param[2 : len(this.Param)-1]
		pieces := strings.SplitN(param, ".", 2)
		prefix := pieces[0]
		if len(pieces) == 1 {
			this.singleParam = ""
		} else {
			this.singleParam = pieces[1]
		}

		if this.checkpointFinder != nil {
			checkpoint := this.checkpointFinder(prefix)
			if checkpoint == nil {
				return errors.New("no check point '" + prefix + "' found")
			}
			this.singleCheckpoint = checkpoint
		} else {
			checkpoint := checkpoints.FindCheckpoint(prefix)
			if checkpoint == nil {
				return errors.New("no check point '" + prefix + "' found")
			}
			checkpoint.Init()
			this.singleCheckpoint = checkpoint
		}

		return nil
	}

	this.multipleCheckpoints = map[string]checkpoints.CheckpointInterface{}
	var err error = nil
	configutils.ParseVariables(this.Param, func(varName string) (value string) {
		pieces := strings.SplitN(varName, ".", 2)
		prefix := pieces[0]
		if this.checkpointFinder != nil {
			checkpoint := this.checkpointFinder(prefix)
			if checkpoint == nil {
				err = errors.New("no check point '" + prefix + "' found")
			} else {
				this.multipleCheckpoints[prefix] = checkpoint
			}
		} else {
			checkpoint := checkpoints.FindCheckpoint(prefix)
			if checkpoint == nil {
				err = errors.New("no check point '" + prefix + "' found")
			} else {
				checkpoint.Init()
				this.multipleCheckpoints[prefix] = checkpoint
			}
		}
		return ""
	})

	return err
}

func (this *Rule) MatchRequest(req requests.Request) (b bool, hasRequestBody bool, err error) {
	if this.singleCheckpoint != nil {
		value, hasCheckedRequestBody, err, _ := this.singleCheckpoint.RequestValue(req, this.singleParam, this.CheckpointOptions, this.Id)
		if hasCheckedRequestBody {
			hasRequestBody = true
		}
		if err != nil {
			return false, hasRequestBody, err
		}

		// execute filters
		if len(this.ParamFilters) > 0 {
			value = this.execFilter(value)
		}

		// if is composed checkpoint, we just returns true or false
		if this.singleCheckpoint.IsComposed() {
			return types.Bool(value), hasRequestBody, nil
		}

		return this.Test(value), hasRequestBody, nil
	}

	value := configutils.ParseVariables(this.Param, func(varName string) (value string) {
		pieces := strings.SplitN(varName, ".", 2)
		prefix := pieces[0]
		point, ok := this.multipleCheckpoints[prefix]
		if !ok {
			return ""
		}

		if len(pieces) == 1 {
			value1, hasCheckRequestBody, err1, _ := point.RequestValue(req, "", this.CheckpointOptions, this.Id)
			if hasCheckRequestBody {
				hasRequestBody = true
			}
			if err1 != nil {
				err = err1
			}
			return types.String(value1)
		}

		value1, hasCheckRequestBody, err1, _ := point.RequestValue(req, pieces[1], this.CheckpointOptions, this.Id)
		if hasCheckRequestBody {
			hasRequestBody = true
		}
		if err1 != nil {
			err = err1
		}
		return types.String(value1)
	})

	if err != nil {
		return false, hasRequestBody, err
	}

	return this.Test(value), hasRequestBody, nil
}

func (this *Rule) MatchResponse(req requests.Request, resp *requests.Response) (b bool, hasRequestBody bool, err error) {
	if this.singleCheckpoint != nil {
		// if is request param
		if this.singleCheckpoint.IsRequest() {
			value, hasCheckRequestBody, err, _ := this.singleCheckpoint.RequestValue(req, this.singleParam, this.CheckpointOptions, this.Id)
			if hasCheckRequestBody {
				hasRequestBody = true
			}
			if err != nil {
				return false, hasRequestBody, err
			}

			// execute filters
			if len(this.ParamFilters) > 0 {
				value = this.execFilter(value)
			}

			return this.Test(value), hasRequestBody, nil
		}

		// response param
		value, hasCheckRequestBody, err, _ := this.singleCheckpoint.ResponseValue(req, resp, this.singleParam, this.CheckpointOptions, this.Id)
		if hasCheckRequestBody {
			hasRequestBody = true
		}
		if err != nil {
			return false, hasRequestBody, err
		}

		// if is composed checkpoint, we just returns true or false
		if this.singleCheckpoint.IsComposed() {
			return types.Bool(value), hasRequestBody, nil
		}

		return this.Test(value), hasRequestBody, nil
	}

	value := configutils.ParseVariables(this.Param, func(varName string) (value string) {
		pieces := strings.SplitN(varName, ".", 2)
		prefix := pieces[0]
		point, ok := this.multipleCheckpoints[prefix]
		if !ok {
			return ""
		}

		if len(pieces) == 1 {
			if point.IsRequest() {
				value1, hasCheckRequestBody, err1, _ := point.RequestValue(req, "", this.CheckpointOptions, this.Id)
				if hasCheckRequestBody {
					hasRequestBody = true
				}
				if err1 != nil {
					err = err1
				}
				return types.String(value1)
			} else {
				value1, hasCheckRequestBody, err1, _ := point.ResponseValue(req, resp, "", this.CheckpointOptions, this.Id)
				if hasCheckRequestBody {
					hasRequestBody = true
				}
				if err1 != nil {
					err = err1
				}
				return types.String(value1)
			}
		}

		if point.IsRequest() {
			value1, hasCheckRequestBody, err1, _ := point.RequestValue(req, pieces[1], this.CheckpointOptions, this.Id)
			if hasCheckRequestBody {
				hasRequestBody = true
			}
			if err1 != nil {
				err = err1
			}
			return types.String(value1)
		} else {
			value1, hasCheckRequestBody, err1, _ := point.ResponseValue(req, resp, pieces[1], this.CheckpointOptions, this.Id)
			if hasCheckRequestBody {
				hasRequestBody = true
			}
			if err1 != nil {
				err = err1
			}
			return types.String(value1)
		}
	})

	if err != nil {
		return false, hasRequestBody, err
	}

	return this.Test(value), hasRequestBody, nil
}

func (this *Rule) Test(value interface{}) bool {
	// operator
	switch this.Operator {
	case RuleOperatorGt:
		return types.Float64(value) > this.floatValue
	case RuleOperatorGte:
		return types.Float64(value) >= this.floatValue
	case RuleOperatorLt:
		return types.Float64(value) < this.floatValue
	case RuleOperatorLte:
		return types.Float64(value) <= this.floatValue
	case RuleOperatorEq:
		return types.Float64(value) == this.floatValue
	case RuleOperatorNeq:
		return types.Float64(value) != this.floatValue
	case RuleOperatorEqString:
		if this.IsCaseInsensitive {
			return strings.ToLower(types.String(value)) == strings.ToLower(this.Value)
		} else {
			return types.String(value) == this.Value
		}
	case RuleOperatorNeqString:
		if this.IsCaseInsensitive {
			return strings.ToLower(types.String(value)) != strings.ToLower(this.Value)
		} else {
			return types.String(value) != this.Value
		}
	case RuleOperatorMatch:
		if value == nil {
			return false
		}

		// strings
		stringList, ok := value.([]string)
		if ok {
			for _, s := range stringList {
				if utils.MatchStringCache(this.reg, s) {
					return true
				}
			}
			return false
		}

		// bytes
		byteSlice, ok := value.([]byte)
		if ok {
			return utils.MatchBytesCache(this.reg, byteSlice)
		}

		// string
		return utils.MatchStringCache(this.reg, types.String(value))
	case RuleOperatorNotMatch:
		if value == nil {
			return true
		}
		stringList, ok := value.([]string)
		if ok {
			for _, s := range stringList {
				if utils.MatchStringCache(this.reg, s) {
					return false
				}
			}
			return true
		}

		// bytes
		byteSlice, ok := value.([]byte)
		if ok {
			return !utils.MatchBytesCache(this.reg, byteSlice)
		}

		return !utils.MatchStringCache(this.reg, types.String(value))
	case RuleOperatorContains:
		if types.IsSlice(value) {
			_, isBytes := value.([]byte)
			if !isBytes {
				ok := false
				lists.Each(value, func(k int, v interface{}) {
					if types.String(v) == this.Value {
						ok = true
					}
				})
				return ok
			}
		}
		if types.IsMap(value) {
			lowerValue := ""
			if this.IsCaseInsensitive {
				lowerValue = strings.ToLower(this.Value)
			}
			for _, v := range maps.NewMap(value) {
				if this.IsCaseInsensitive {
					if strings.ToLower(types.String(v)) == lowerValue {
						return true
					}
				} else {
					if types.String(v) == this.Value {
						return true
					}
				}
			}
			return false
		}

		if this.IsCaseInsensitive {
			return strings.Contains(strings.ToLower(types.String(value)), strings.ToLower(this.Value))
		} else {
			return strings.Contains(types.String(value), this.Value)
		}
	case RuleOperatorNotContains:
		if this.IsCaseInsensitive {
			return !strings.Contains(strings.ToLower(types.String(value)), strings.ToLower(this.Value))
		} else {
			return !strings.Contains(types.String(value), this.Value)
		}
	case RuleOperatorPrefix:
		if this.IsCaseInsensitive {
			return strings.HasPrefix(strings.ToLower(types.String(value)), strings.ToLower(this.Value))
		} else {
			return strings.HasPrefix(types.String(value), this.Value)
		}
	case RuleOperatorSuffix:
		if this.IsCaseInsensitive {
			return strings.HasSuffix(strings.ToLower(types.String(value)), strings.ToLower(this.Value))
		} else {
			return strings.HasSuffix(types.String(value), this.Value)
		}
	case RuleOperatorContainsBinary:
		data, _ := base64.StdEncoding.DecodeString(types.String(this.Value))
		if this.IsCaseInsensitive {
			return bytes.Contains(bytes.ToUpper([]byte(types.String(value))), bytes.ToUpper(data))
		} else {
			return bytes.Contains([]byte(types.String(value)), data)
		}
	case RuleOperatorNotContainsBinary:
		data, _ := base64.StdEncoding.DecodeString(types.String(this.Value))
		if this.IsCaseInsensitive {
			return !bytes.Contains(bytes.ToUpper([]byte(types.String(value))), bytes.ToUpper(data))
		} else {
			return !bytes.Contains([]byte(types.String(value)), data)
		}
	case RuleOperatorHasKey:
		if types.IsSlice(value) {
			index := types.Int(this.Value)
			if index < 0 {
				return false
			}
			return reflect.ValueOf(value).Len() > index
		} else if types.IsMap(value) {
			m := maps.NewMap(value)
			if this.IsCaseInsensitive {
				lowerValue := strings.ToLower(this.Value)
				for k := range m {
					if strings.ToLower(k) == lowerValue {
						return true
					}
				}
			} else {
				return m.Has(this.Value)
			}
		} else {
			return false
		}

	case RuleOperatorVersionGt:
		return stringutil.VersionCompare(this.Value, types.String(value)) > 0
	case RuleOperatorVersionLt:
		return stringutil.VersionCompare(this.Value, types.String(value)) < 0
	case RuleOperatorVersionRange:
		if strings.Contains(this.Value, ",") {
			versions := strings.SplitN(this.Value, ",", 2)
			version1 := strings.TrimSpace(versions[0])
			version2 := strings.TrimSpace(versions[1])
			if len(version1) > 0 && stringutil.VersionCompare(types.String(value), version1) < 0 {
				return false
			}
			if len(version2) > 0 && stringutil.VersionCompare(types.String(value), version2) > 0 {
				return false
			}
			return true
		} else {
			return stringutil.VersionCompare(types.String(value), this.Value) >= 0
		}
	case RuleOperatorEqIP:
		ip := net.ParseIP(types.String(value))
		if ip == nil {
			return false
		}
		return this.isIP && bytes.Compare(this.ipValue, ip) == 0
	case RuleOperatorGtIP:
		ip := net.ParseIP(types.String(value))
		if ip == nil {
			return false
		}
		return this.isIP && bytes.Compare(ip, this.ipValue) > 0
	case RuleOperatorGteIP:
		ip := net.ParseIP(types.String(value))
		if ip == nil {
			return false
		}
		return this.isIP && bytes.Compare(ip, this.ipValue) >= 0
	case RuleOperatorLtIP:
		ip := net.ParseIP(types.String(value))
		if ip == nil {
			return false
		}
		return this.isIP && bytes.Compare(ip, this.ipValue) < 0
	case RuleOperatorLteIP:
		ip := net.ParseIP(types.String(value))
		if ip == nil {
			return false
		}
		return this.isIP && bytes.Compare(ip, this.ipValue) <= 0
	case RuleOperatorIPRange:
		return this.containsIP(value)
	case RuleOperatorNotIPRange:
		return !this.containsIP(value)
	case RuleOperatorIPMod:
		pieces := strings.SplitN(this.Value, ",", 2)
		if len(pieces) == 1 {
			rem := types.Int64(pieces[0])
			return this.ipToInt64(net.ParseIP(types.String(value)))%10 == rem
		}
		div := types.Int64(pieces[0])
		if div == 0 {
			return false
		}
		rem := types.Int64(pieces[1])
		return this.ipToInt64(net.ParseIP(types.String(value)))%div == rem
	case RuleOperatorIPMod10:
		return this.ipToInt64(net.ParseIP(types.String(value)))%10 == types.Int64(this.Value)
	case RuleOperatorIPMod100:
		return this.ipToInt64(net.ParseIP(types.String(value)))%100 == types.Int64(this.Value)
	}
	return false
}

func (this *Rule) IsSingleCheckpoint() bool {
	return this.singleCheckpoint != nil
}

func (this *Rule) SetCheckpointFinder(finder func(prefix string) checkpoints.CheckpointInterface) {
	this.checkpointFinder = finder
}

func (this *Rule) unescape(v string) string {
	//replace urlencoded characters
	v = strings.Replace(v, `\s`, `(\s|%09|%0A|\+)`, -1)
	v = strings.Replace(v, `\(`, `(\(|%28)`, -1)
	v = strings.Replace(v, `=`, `(=|%3D)`, -1)
	v = strings.Replace(v, `<`, `(<|%3C)`, -1)
	v = strings.Replace(v, `\*`, `(\*|%2A)`, -1)
	v = strings.Replace(v, `\\`, `(\\|%2F)`, -1)
	v = strings.Replace(v, `!`, `(!|%21)`, -1)
	v = strings.Replace(v, `/`, `(/|%2F)`, -1)
	v = strings.Replace(v, `;`, `(;|%3B)`, -1)
	v = strings.Replace(v, `\+`, `(\+|%20)`, -1)
	return v
}

func (this *Rule) containsIP(value interface{}) bool {
	ip := net.ParseIP(types.String(value))
	if ip == nil {
		return false
	}

	// 检查IP范围格式
	if strings.Contains(this.Value, ",") {
		ipList := strings.SplitN(this.Value, ",", 2)
		ipString1 := strings.TrimSpace(ipList[0])
		ipString2 := strings.TrimSpace(ipList[1])

		if len(ipString1) > 0 {
			ip1 := net.ParseIP(ipString1)
			if ip1 == nil {
				return false
			}

			if bytes.Compare(ip, ip1) < 0 {
				return false
			}
		}

		if len(ipString2) > 0 {
			ip2 := net.ParseIP(ipString2)
			if ip2 == nil {
				return false
			}

			if bytes.Compare(ip, ip2) > 0 {
				return false
			}
		}

		return true
	} else if strings.Contains(this.Value, "/") {
		_, ipNet, err := net.ParseCIDR(this.Value)
		if err != nil {
			return false
		}
		return ipNet.Contains(ip)
	} else {
		return false
	}
}

func (this *Rule) ipToInt64(ip net.IP) int64 {
	if len(ip) == 0 {
		return 0
	}
	if len(ip) == 16 {
		return int64(binary.BigEndian.Uint32(ip[12:16]))
	}
	return int64(binary.BigEndian.Uint32(ip))
}

func (this *Rule) execFilter(value interface{}) interface{} {
	var goNext bool
	var err error

	for _, filter := range this.ParamFilters {
		filterInstance := filterconfigs.FindFilter(filter.Code)
		if filterInstance == nil {
			continue
		}
		value, goNext, err = filterInstance.Do(value, filter.Options)
		if err != nil {
			remotelogs.Error("WAF", "filter error: "+err.Error())
			break
		}
		if !goNext {
			break
		}
	}
	return value
}

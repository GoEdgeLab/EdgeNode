package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"github.com/iwind/TeaGo/maps"
)

// CheckpointInterface Check Point
type CheckpointInterface interface {
	// Init initialize
	Init()

	// IsRequest is request?
	IsRequest() bool

	// IsComposed is composed?
	IsComposed() bool

	// RequestValue get request value
	RequestValue(req requests.Request, param string, options maps.Map) (value interface{}, hasRequestBody bool, sysErr error, userErr error)

	// ResponseValue get response value
	ResponseValue(req requests.Request, resp *requests.Response, param string, options maps.Map) (value interface{}, hasRequestBody bool, sysErr error, userErr error)

	// ParamOptions param option list
	ParamOptions() *ParamOptions

	// Options options
	Options() []OptionInterface

	// Start start
	Start()

	// Stop stop
	Stop()
}

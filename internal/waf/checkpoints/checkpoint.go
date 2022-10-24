package checkpoints

import (
	"github.com/TeaOSLab/EdgeNode/internal/waf/requests"
	"net/http"
)

type Checkpoint struct {
	priority int
}

func (this *Checkpoint) Init() {

}

func (this *Checkpoint) IsRequest() bool {
	return true
}

func (this *Checkpoint) IsComposed() bool {
	return false
}

func (this *Checkpoint) ParamOptions() *ParamOptions {
	return nil
}

func (this *Checkpoint) Options() []OptionInterface {
	return nil
}

func (this *Checkpoint) Start() {

}

func (this *Checkpoint) Stop() {

}

func (this *Checkpoint) SetPriority(priority int) {
	this.priority = priority
}

func (this *Checkpoint) Priority() int {
	return this.priority
}

func (this *Checkpoint) RequestBodyIsEmpty(req requests.Request) bool {
	if req.WAFRaw().ContentLength == 0 {
		return true
	}

	var method = req.WAFRaw().Method
	if method == http.MethodHead || method == http.MethodGet {
		return true
	}

	return false
}

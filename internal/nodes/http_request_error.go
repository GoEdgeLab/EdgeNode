package nodes

import (
	"github.com/iwind/TeaGo/types"
	"net/http"
)

func (this *HTTPRequest) write404() {
	if this.doPage(http.StatusNotFound) {
		return
	}

	this.processResponseHeaders(http.StatusNotFound)

	msg := "404 page not found: '" + this.RawURI() + "'"

	this.writer.WriteHeader(http.StatusNotFound)
	_, _ = this.writer.Write([]byte(msg))
}

func (this *HTTPRequest) write50x(err error, statusCode int) {
	if err != nil {
		this.addError(err)
	}

	if this.doPage(statusCode) {
		return
	}
	this.processResponseHeaders(statusCode)
	this.writer.WriteHeader(statusCode)
	_, _ = this.writer.Write([]byte(types.String(statusCode) + " " + http.StatusText(statusCode)))
}

package nodes

import (
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

func (this *HTTPRequest) write500(err error) {
	if err != nil {
		this.addError(err)
	}

	statusCode := http.StatusInternalServerError
	if this.doPage(statusCode) {
		return
	}
	this.processResponseHeaders(statusCode)
	this.writer.WriteHeader(statusCode)
	_, _ = this.writer.Write([]byte(http.StatusText(statusCode)))
}

func (this *HTTPRequest) write502(err error) {
	if err != nil {
		this.addError(err)
	}

	statusCode := http.StatusBadGateway
	if this.doPage(statusCode) {
		return
	}
	this.processResponseHeaders(statusCode)
	this.writer.WriteHeader(statusCode)
	_, _ = this.writer.Write([]byte("502 Bad Gateway"))
}

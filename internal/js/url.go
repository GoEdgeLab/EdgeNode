// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package js

import (
	"github.com/dop251/goja"
	"github.com/iwind/TeaGo/types"
	"net/url"
)

type URL struct {
	u *url.URL
}

func NewURL(u *url.URL) *URL {
	return &URL{
		u: u,
	}
}

func (this *URL) JSNew(args []goja.Value) *URL {
	var urlString = ""
	if len(args) == 1 {
		urlString = args[0].String()
	}
	u, _ := url.Parse(urlString)
	if u == nil {
		u = &url.URL{}
	}
	return NewURL(u)
}

func (this *URL) Port() int {
	return types.Int(this.u.Port())
}

func (this *URL) Args() map[string][]string {
	return this.u.Query()
}

func (this *URL) Arg(name string) string {
	return this.u.Query().Get(name)
}

func (this *URL) Username() string {
	if this.u.User != nil {
		return this.u.User.Username()
	}
	return ""
}

func (this *URL) Password() string {
	if this.u.User != nil {
		password, _ := this.u.User.Password()
		return password
	}
	return ""
}

func (this *URL) Uri() string {
	return this.u.RequestURI()
}

func (this *URL) Path() string {
	return this.u.Path
}

func (this *URL) Host() string {
	return this.u.Host
}

func (this *URL) Fragment() string {
	return this.u.Fragment
}

func (this *URL) Hash() string {
	if len(this.u.Fragment) > 0 {
		return "#" + this.u.Fragment
	} else {
		return ""
	}
}

func (this *URL) Scheme() string {
	return this.u.Scheme
}

func (this *URL) String() string {
	return this.u.String()
}

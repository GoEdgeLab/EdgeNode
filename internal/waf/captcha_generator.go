// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package waf

import (
	"bytes"
	"github.com/dchest/captcha"
	"github.com/iwind/TeaGo/rands"
	"io"
	"time"
)

// CaptchaGenerator captcha generator
type CaptchaGenerator struct {
	store captcha.Store
}

func NewCaptchaGenerator() *CaptchaGenerator {
	return &CaptchaGenerator{
		store: captcha.NewMemoryStore(100_000, 5*time.Minute),
	}
}

// NewCaptcha create new captcha
func (this *CaptchaGenerator) NewCaptcha(length int) (captchaId string) {
	captchaId = rands.HexString(16)

	if length <= 0 || length > 20 {
		length = 4
	}

	this.store.Set(captchaId, captcha.RandomDigits(length))
	return
}

// WriteImage write image to front writer
func (this *CaptchaGenerator) WriteImage(w io.Writer, id string, width, height int) error {
	var d = this.store.Get(id, false)
	if d == nil {
		return captcha.ErrNotFound
	}
	_, err := captcha.NewImage(id, d, width, height).WriteTo(w)
	return err
}

// Verify user input
func (this *CaptchaGenerator) Verify(id string, digits string) bool {
	var countDigits = len(digits)
	if countDigits == 0 {
		return false
	}
	var value = this.store.Get(id, true)
	if len(value) != countDigits {
		return false
	}

	var nb = make([]byte, countDigits)
	for i := 0; i < countDigits; i++ {
		var d = digits[i]
		if d >= '0' && d <= '9' {
			nb[i] = d - '0'
		}
	}

	return bytes.Equal(nb, value)
}

// Get captcha data
func (this *CaptchaGenerator) Get(id string) []byte {
	return this.store.Get(id, false)
}

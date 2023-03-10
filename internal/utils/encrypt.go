// Copyright 2021 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/nodeconfigs"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	"github.com/TeaOSLab/EdgeNode/internal/events"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/rands"
	stringutil "github.com/iwind/TeaGo/utils/string"
)

var (
	simpleEncryptMagicKey = rands.HexString(32)
)

func init() {
	if !teaconst.IsMain {
		return
	}

	events.On(events.EventReload, func() {
		nodeConfig, _ := nodeconfigs.SharedNodeConfig()
		if nodeConfig != nil {
			simpleEncryptMagicKey = stringutil.Md5(nodeConfig.NodeId + "@" + nodeConfig.Secret)
		}
	})
}

// SimpleEncrypt 加密特殊信息
func SimpleEncrypt(data []byte) []byte {
	var method = &AES256CFBMethod{}
	err := method.Init([]byte(simpleEncryptMagicKey), []byte(simpleEncryptMagicKey[:16]))
	if err != nil {
		logs.Println("[SimpleEncrypt]" + err.Error())
		return data
	}

	dst, err := method.Encrypt(data)
	if err != nil {
		logs.Println("[SimpleEncrypt]" + err.Error())
		return data
	}
	return dst
}

// SimpleDecrypt 解密特殊信息
func SimpleDecrypt(data []byte) []byte {
	var method = &AES256CFBMethod{}
	err := method.Init([]byte(simpleEncryptMagicKey), []byte(simpleEncryptMagicKey[:16]))
	if err != nil {
		logs.Println("[MagicKeyEncode]" + err.Error())
		return data
	}

	src, err := method.Decrypt(data)
	if err != nil {
		logs.Println("[MagicKeyEncode]" + err.Error())
		return data
	}
	return src
}

func SimpleEncryptMap(m maps.Map) (base64String string, err error) {
	mJSON, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	data := SimpleEncrypt(mJSON)
	return base64.StdEncoding.EncodeToString(data), nil
}

func SimpleDecryptMap(base64String string) (maps.Map, error) {
	data, err := base64.StdEncoding.DecodeString(base64String)
	if err != nil {
		return nil, err
	}
	mJSON := SimpleDecrypt(data)
	var result = maps.Map{}
	err = json.Unmarshal(mJSON, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type AES256CFBMethod struct {
	block cipher.Block
	iv    []byte
}

func (this *AES256CFBMethod) Init(key, iv []byte) error {
	// 判断key是否为32长度
	l := len(key)
	if l > 32 {
		key = key[:32]
	} else if l < 32 {
		key = append(key, bytes.Repeat([]byte{' '}, 32-l)...)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	this.block = block

	// 判断iv长度
	l2 := len(iv)
	if l2 > aes.BlockSize {
		iv = iv[:aes.BlockSize]
	} else if l2 < aes.BlockSize {
		iv = append(iv, bytes.Repeat([]byte{' '}, aes.BlockSize-l2)...)
	}
	this.iv = iv

	return nil
}

func (this *AES256CFBMethod) Encrypt(src []byte) (dst []byte, err error) {
	if len(src) == 0 {
		return
	}

	defer func() {
		r := recover()
		if r != nil {
			err = errors.New("encrypt failed")
		}
	}()

	dst = make([]byte, len(src))

	encrypter := cipher.NewCFBEncrypter(this.block, this.iv)
	encrypter.XORKeyStream(dst, src)

	return
}

func (this *AES256CFBMethod) Decrypt(dst []byte) (src []byte, err error) {
	if len(dst) == 0 {
		return
	}

	defer func() {
		r := recover()
		if r != nil {
			err = errors.New("decrypt failed")
		}
	}()

	src = make([]byte, len(dst))
	decrypter := cipher.NewCFBDecrypter(this.block, this.iv)
	decrypter.XORKeyStream(src, dst)

	return
}

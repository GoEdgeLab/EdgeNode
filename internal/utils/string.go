package utils

import (
	"strings"
	"unsafe"
)

// UnsafeBytesToString convert bytes to string
func UnsafeBytesToString(bs []byte) string {
	return *(*string)(unsafe.Pointer(&bs))
}

// UnsafeStringToBytes convert string to bytes
func UnsafeStringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&s))
}

// FormatAddress format address
func FormatAddress(addr string) string {
	if strings.HasSuffix(addr, "unix:") {
		return addr
	}
	addr = strings.Replace(addr, " ", "", -1)
	addr = strings.Replace(addr, "\t", "", -1)
	addr = strings.Replace(addr, "：", ":", -1)
	addr = strings.TrimSpace(addr)
	return addr
}

// FormatAddressList format address list
func FormatAddressList(addrList []string) []string {
	result := []string{}
	for _, addr := range addrList {
		result = append(result, FormatAddress(addr))
	}
	return result
}

func ToValidUTF8string(v string) string {
	return strings.ToValidUTF8(v, "")
}

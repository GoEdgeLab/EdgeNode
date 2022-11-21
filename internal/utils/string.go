package utils

import (
	"sort"
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

// ToValidUTF8string 去除字符串中的非UTF-8字符
func ToValidUTF8string(v string) string {
	return strings.ToValidUTF8(v, "")
}

// EqualStrings 检查两个字符串slice内容是否一致
func EqualStrings(s1 []string, s2 []string) bool {
	if len(s1) != len(s2) {
		return false
	}
	sort.Strings(s1)
	sort.Strings(s2)
	for index, v1 := range s1 {
		if v1 != s2[index] {
			return false
		}
	}
	return true
}

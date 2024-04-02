// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package byteutils

// Copy bytes
func Copy(b []byte) []byte {
	var l = len(b)
	if l == 0 {
		return []byte{}
	}
	var d = make([]byte, l)
	copy(d, b)
	return d
}

// Append bytes
func Append(b []byte, b2 ...byte) []byte {
	return append(Copy(b), b2...)
}

// Concat bytes
func Concat(b []byte, b2 ...[]byte) []byte {
	b = Copy(b)
	for _, b3 := range b2 {
		b = append(b, b3...)
	}
	return b
}

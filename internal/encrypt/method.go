package encrypt

type MethodInterface interface {
	// Init 初始化
	Init(key []byte, iv []byte) error

	// Encrypt 加密
	Encrypt(src []byte) (dst []byte, err error)

	// Decrypt 解密
	Decrypt(dst []byte) (src []byte, err error)
}

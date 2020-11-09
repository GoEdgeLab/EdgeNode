package iplibrary

type LibraryInterface interface {
	// 加载数据库文件
	Load(dbPath string) error

	// 查询IP
	Lookup(ip string) (*Result, error)

	// 关闭数据库文件
	Close()
}

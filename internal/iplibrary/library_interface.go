package iplibrary

type LibraryInterface interface {
	// Load 加载数据库文件
	Load(dbPath string) error

	// Lookup 查询IP
	// 返回结果有可能为空
	Lookup(ip string) (*Result, error)

	// Close 关闭数据库文件
	Close()
}

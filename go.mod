module github.com/TeaOSLab/EdgeNode

go 1.15

replace github.com/TeaOSLab/EdgeCommon => ../EdgeCommon

require (
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/TeaOSLab/EdgeCommon v0.0.0-00010101000000-000000000000
	github.com/andybalholm/brotli v1.0.3
	github.com/cespare/xxhash v1.1.0
	github.com/chai2010/webp v1.1.0
	github.com/dchest/captcha v0.0.0-20200903113550-03f5f0333e1f
	github.com/dop251/goja v0.0.0-20210804101310-32956a348b49
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/go-yaml/yaml v2.1.0+incompatible
	github.com/golang/protobuf v1.5.2
	github.com/iwind/TeaGo v0.0.0-20210628135026-38575a4ab060
	github.com/iwind/gofcgi v0.0.0-20210528023741-a92711d45f11
	github.com/iwind/gosock v0.0.0-20210722083328-12b2d66abec3
	github.com/lionsoul2014/ip2region v2.2.0-release+incompatible
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/mssola/user_agent v0.5.2
	github.com/shirou/gopsutil v3.21.5+incompatible
	github.com/tklauser/go-sysconf v0.3.6 // indirect
	golang.org/x/image v0.0.0-20190802002840-cff245a6509b
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e
	golang.org/x/sys v0.0.0-20210616094352-59db8d763f22
	golang.org/x/text v0.3.6
	google.golang.org/genproto v0.0.0-20210617175327-b9e0b3197ced // indirect
	google.golang.org/grpc v1.38.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
)

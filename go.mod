module github.com/TeaOSLab/EdgeNode

go 1.15

replace github.com/TeaOSLab/EdgeCommon => ../EdgeCommon

require (
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/TeaOSLab/EdgeCommon v0.0.0-00010101000000-000000000000
	github.com/cespare/xxhash v1.1.0
	github.com/dchest/captcha v0.0.0-20200903113550-03f5f0333e1f
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/go-yaml/yaml v2.1.0+incompatible
	github.com/golang/protobuf v1.4.2
	github.com/iwind/TeaGo v0.0.0-20210411134150-ddf57e240c2f
	github.com/iwind/gofcgi v0.0.0-20210528023741-a92711d45f11
	github.com/lionsoul2014/ip2region v2.2.0-release+incompatible
	github.com/mattn/go-sqlite3 v1.14.7
	github.com/mssola/user_agent v0.5.2
	github.com/shirou/gopsutil v2.20.9+incompatible
	golang.org/x/net v0.0.0-20200520004742-59133d7f0dd7
	golang.org/x/sys v0.0.0-20200519105757-fe76b779f299
	golang.org/x/text v0.3.2
	google.golang.org/grpc v1.32.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
)

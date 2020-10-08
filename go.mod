module github.com/TeaOSLab/EdgeNode

go 1.15

replace github.com/TeaOSLab/EdgeCommon => ../EdgeCommon

require (
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/TeaOSLab/EdgeCommon v0.0.0-00010101000000-000000000000
	github.com/dchest/captcha v0.0.0-20200903113550-03f5f0333e1f
	github.com/dchest/siphash v1.2.1
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/go-yaml/yaml v2.1.0+incompatible
	github.com/iwind/TeaGo v0.0.0-20200923021120-f5d76441fe9e
	github.com/shirou/gopsutil v2.20.9+incompatible
	golang.org/x/net v0.0.0-20200520004742-59133d7f0dd7
	google.golang.org/grpc v1.32.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
)

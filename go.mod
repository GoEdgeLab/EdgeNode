module github.com/TeaOSLab/EdgeNode

go 1.15

replace (
	github.com/TeaOSLab/EdgeCommon => ../EdgeCommon
)

require (
	github.com/TeaOSLab/EdgeCommon v0.0.0-00010101000000-000000000000
	github.com/andybalholm/brotli v1.0.4
	github.com/biessek/golang-ico v0.0.0-20180326222316-d348d9ea4670
	github.com/cespare/xxhash v1.1.0
	github.com/chai2010/webp v1.1.0 // indirect
	github.com/dchest/captcha v0.0.0-20200903113550-03f5f0333e1f
	github.com/fsnotify/fsnotify v1.5.1
	github.com/golang/protobuf v1.5.2
	github.com/iwind/TeaGo v0.0.0-20220304043459-0dd944a5b475
	github.com/iwind/gofcgi v0.0.0-20210528023741-a92711d45f11
	github.com/iwind/gosock v0.0.0-20210722083328-12b2d66abec3
	github.com/iwind/gowebp v0.0.0-20211029040624-7331ecc78ed8
	github.com/jsummers/gobmp v0.0.0-20151104160322-e2ba15ffa76e // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mattn/go-sqlite3 v1.14.9
	github.com/miekg/dns v1.1.43
	github.com/mssola/user_agent v0.5.3
	github.com/pires/go-proxyproto v0.6.1
	github.com/shirou/gopsutil/v3 v3.22.2
	golang.org/x/image v0.0.0-20211028202545-6944b10bf410
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f
	golang.org/x/sys v0.0.0-20220227234510-4e6760a101f9
	golang.org/x/text v0.3.7
	google.golang.org/grpc v1.44.0
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)

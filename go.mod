module github.com/TeaOSLab/EdgeNode

go 1.18

replace (
	github.com/TeaOSLab/EdgeCommon => ../EdgeCommon
)

require (
	github.com/TeaOSLab/EdgeCommon v0.0.0-00010101000000-000000000000
	github.com/andybalholm/brotli v1.0.4
	github.com/biessek/golang-ico v0.0.0-20180326222316-d348d9ea4670
	github.com/cespare/xxhash v1.1.0
	github.com/dchest/captcha v0.0.0-20200903113550-03f5f0333e1f
	github.com/fsnotify/fsnotify v1.5.1
	github.com/go-redis/redis/v8 v8.11.5
	github.com/golang/protobuf v1.5.2
	github.com/google/nftables v0.0.0-20220407195405-950e408d48c6
	github.com/iwind/TeaGo v0.0.0-20220304043459-0dd944a5b475
	github.com/iwind/gofcgi v0.0.0-20210528023741-a92711d45f11
	github.com/iwind/gosock v0.0.0-20211103081026-ee4652210ca4
	github.com/iwind/gowebp v0.0.0-20211029040624-7331ecc78ed8
	github.com/klauspost/compress v1.15.8
	github.com/mattn/go-sqlite3 v1.14.9
	github.com/mdlayher/netlink v1.4.2
	github.com/miekg/dns v1.1.43
	github.com/mssola/user_agent v0.5.3
	github.com/pires/go-proxyproto v0.6.1
	github.com/shirou/gopsutil/v3 v3.22.2
	golang.org/x/image v0.0.0-20211028202545-6944b10bf410
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f
	golang.org/x/sys v0.0.0-20220412211240-33da011f77ad
	golang.org/x/text v0.3.7
	google.golang.org/grpc v1.45.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)

require (
	github.com/BurntSushi/toml v0.4.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/chai2010/webp v1.1.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/google/go-cmp v0.5.7 // indirect
	github.com/josharian/native v0.0.0-20200817173448-b6b71def0850 // indirect
	github.com/jsummers/gobmp v0.0.0-20151104160322-e2ba15ffa76e // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/mdlayher/socket v0.0.0-20211102153432-57e3fa563ecb // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/tklauser/go-sysconf v0.3.9 // indirect
	github.com/tklauser/numcpus v0.3.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	golang.org/x/mod v0.5.1 // indirect
	golang.org/x/tools v0.1.8 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/genproto v0.0.0-20220317150908-0efb43f6373e // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	honnef.co/go/tools v0.2.2 // indirect
)

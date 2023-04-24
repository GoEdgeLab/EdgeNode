module github.com/TeaOSLab/EdgeNode

go 1.18

replace (
	github.com/TeaOSLab/EdgeCommon => ../EdgeCommon
	github.com/fsnotify/fsnotify => github.com/iwind/fsnotify v1.5.2-0.20220817040843-193be2051ff4
	github.com/google/nftables => github.com/iwind/nftables v0.0.0-20230419014751-9f023a644ad4
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
	github.com/google/nftables v0.1.0
	github.com/iwind/TeaGo v0.0.0-20230304012706-c1f4a4e27470
	github.com/iwind/gofcgi v0.0.0-20210528023741-a92711d45f11
	github.com/iwind/gosock v0.0.0-20211103081026-ee4652210ca4
	github.com/iwind/gowebp v0.0.0-20220819053541-c235395277b5
	github.com/klauspost/compress v1.16.5
	github.com/mattn/go-sqlite3 v1.14.9
	github.com/mdlayher/netlink v1.7.1
	github.com/miekg/dns v1.1.43
	github.com/mssola/user_agent v0.5.3
	github.com/pires/go-proxyproto v0.6.1
	github.com/shirou/gopsutil/v3 v3.22.2
	golang.org/x/image v0.0.0-20220722155232-062f8c9fd539
	golang.org/x/net v0.8.0
	golang.org/x/sys v0.6.0
	golang.org/x/text v0.8.0
	google.golang.org/grpc v1.45.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/chai2010/webp v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/josharian/native v1.0.0 // indirect
	github.com/jsummers/gobmp v0.0.0-20151104160322-e2ba15ffa76e // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/mdlayher/socket v0.4.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/tklauser/go-sysconf v0.3.9 // indirect
	github.com/tklauser/numcpus v0.3.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	golang.org/x/sync v0.1.0 // indirect
	google.golang.org/genproto v0.0.0-20220317150908-0efb43f6373e // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

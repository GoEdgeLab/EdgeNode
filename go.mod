module github.com/TeaOSLab/EdgeNode

go 1.18

replace (
	github.com/TeaOSLab/EdgeCommon => ../EdgeCommon
	github.com/fsnotify/fsnotify => github.com/iwind/fsnotify v1.5.2-0.20220817040843-193be2051ff4
	github.com/google/nftables => github.com/iwind/nftables v0.0.0-20230419014751-9f023a644ad4
)

require (
	github.com/TeaOSLab/EdgeCommon v0.0.0-00010101000000-000000000000
	github.com/aliyun/aliyun-oss-go-sdk v2.2.7+incompatible
	github.com/andybalholm/brotli v1.0.5
	github.com/aws/aws-sdk-go v1.44.279
	github.com/baidubce/bce-sdk-go v0.9.153
	github.com/biessek/golang-ico v0.0.0-20180326222316-d348d9ea4670
	github.com/cespare/xxhash v1.1.0
	github.com/dchest/captcha v0.0.0-20200903113550-03f5f0333e1f
	github.com/fsnotify/fsnotify v1.6.0
	github.com/go-redis/redis/v8 v8.11.5
	github.com/google/gopacket v1.1.19
	github.com/google/nftables v0.1.0
	github.com/huaweicloud/huaweicloud-sdk-go-obs v3.23.4+incompatible
	github.com/iwind/TeaGo v0.0.0-20230630104525-161f0b32996d
	github.com/iwind/gofcgi v0.0.0-20210528023741-a92711d45f11
	github.com/iwind/gosock v0.0.0-20211103081026-ee4652210ca4
	github.com/iwind/gowebp v0.0.0-20231026013903-1c22b0d78cc4
	github.com/klauspost/compress v1.17.2
	github.com/mattn/go-sqlite3 v1.14.17
	github.com/mdlayher/netlink v1.7.1
	github.com/miekg/dns v1.1.43
	github.com/mssola/useragent v1.0.0
	github.com/pires/go-proxyproto v0.6.1
	github.com/qiniu/go-sdk/v7 v7.16.0
	github.com/quic-go/quic-go v0.39.2
	github.com/shirou/gopsutil/v3 v3.22.2
	github.com/tencentyun/cos-go-sdk-v5 v0.7.41
	golang.org/x/image v0.13.0
	golang.org/x/net v0.17.0
	golang.org/x/sys v0.13.0
	google.golang.org/grpc v1.59.0
	google.golang.org/protobuf v1.31.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/clbanning/mxj v1.8.4 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/google/pprof v0.0.0-20231023181126-ff6d637d2a7b // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/native v1.0.0 // indirect
	github.com/jsummers/gobmp v0.0.0-20151104160322-e2ba15ffa76e // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/mdlayher/socket v0.4.0 // indirect
	github.com/mitchellh/mapstructure v1.4.3 // indirect
	github.com/mozillazg/go-httpheader v0.2.1 // indirect
	github.com/onsi/ginkgo/v2 v2.13.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/quic-go/qpack v0.4.0 // indirect
	github.com/quic-go/qtls-go1-20 v0.4.1 // indirect
	github.com/tdewolff/minify/v2 v2.12.7 // indirect
	github.com/tdewolff/parse/v2 v2.6.6 // indirect
	github.com/tklauser/go-sysconf v0.3.9 // indirect
	github.com/tklauser/numcpus v0.3.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	go.uber.org/mock v0.3.0 // indirect
	golang.org/x/crypto v0.14.0 // indirect
	golang.org/x/exp v0.0.0-20231006140011-7918f672742d // indirect
	golang.org/x/mod v0.13.0 // indirect
	golang.org/x/sync v0.4.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	golang.org/x/tools v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231016165738-49dd2c1f3d0b // indirect
)

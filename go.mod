module github.com/xtls/xray-core

go 1.27

replace github.com/xtls/reality => github.com/domaingts/electricity v0.2.0

require (
	github.com/cloudflare/circl v1.6.5
	github.com/golang/mock v1.7.0-rc.1
	github.com/google/go-cmp v0.7.0
	github.com/klauspost/cpuid/v2 v2.4.0
	github.com/miekg/dns v1.1.73
	github.com/quic-go/quic-go v0.62.0
	github.com/refraction-networking/utls v1.8.2
	github.com/sagernet/sing v0.9.0
	github.com/sagernet/sing-shadowsocks v0.2.9
	github.com/stretchr/testify v1.12.1
	github.com/xtls/reality v0.0.0-20260322125925-9234c772ba8f
	go4.org/netipx v0.0.0-20260823151212-3075585bcbeb
	golang.org/x/crypto v0.56.0
	golang.org/x/net v0.58.0
	golang.org/x/sync v0.22.0
	golang.org/x/sys v0.47.0
	google.golang.org/protobuf v1.36.12
	lukechampine.com/blake3 v1.4.1
)

require (
	github.com/andybalholm/brotli v1.0.6 // indirect
	github.com/juju/ratelimit v1.0.2 // indirect
	github.com/klauspost/compress v1.17.4 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/quic-go/qpack v0.6.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/text v0.41.0 // indirect
)

module file-transfer

go 1.24

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/grandcat/zeroconf v1.0.0
	github.com/pion/stun v0.6.1
	github.com/quic-go/quic-go v0.59.0
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
)

require (
	file-transfer/backend v0.0.0-00010101000000-000000000000 // indirect
	github.com/bytedance/sonic v1.9.1 // indirect
	github.com/chenzhuoyu/base64x v0.0.0-20221115062448-fe3a3abad311 // indirect
	github.com/gabriel-vasile/mimetype v1.4.2 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.14.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.2.4 // indirect
	github.com/leodido/go-urn v1.2.4 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.0.8 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.11 // indirect
	golang.org/x/arch v0.3.0 // indirect
	golang.org/x/crypto v0.41.0 // indirect
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	file-transfer/backend => ./backend
	file-transfer/pkg/xfer => ./pkg/xfer
	file-transfer/pkg/xfer/discovery => ./pkg/xfer/discovery
	file-transfer/pkg/xfer/signaling => ./pkg/xfer/signaling
	file-transfer/pkg/xfer/stun => ./pkg/xfer/stun
	file-transfer/pkg/xfer/util => ./pkg/xfer/util
)

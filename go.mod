module github.com/beeper/feedserv

go 1.19

require (
	github.com/rs/zerolog v1.29.0
	go.mau.fi/zeroconfig v0.1.2
	golang.org/x/net v0.8.0
	gopkg.in/yaml.v3 v3.0.1
	maunium.net/go/mautrix v0.15.0
)

require (
	github.com/coreos/go-systemd/v22 v22.3.3-0.20220203105225-a9a7ef127534 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/tidwall/gjson v1.14.4 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	golang.org/x/crypto v0.7.0 // indirect
	golang.org/x/sys v0.6.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	maunium.net/go/maulogger/v2 v2.4.1 // indirect
)

replace maunium.net/go/mautrix => ../../Matrix/mautrix-go

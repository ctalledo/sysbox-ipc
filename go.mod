module github.com/nestybox/sysbox-ipc

go 1.13

require (
	github.com/golang/protobuf v1.3.1
	github.com/opencontainers/runc v0.0.0-00010101000000-000000000000
	github.com/opencontainers/runtime-spec v0.1.2-0.20190207185410-29686dbc5559
	google.golang.org/grpc v1.21.0
)

replace github.com/opencontainers/runc => ../sysbox-runc

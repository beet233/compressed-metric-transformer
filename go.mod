module main

go 1.20

require (
	github.com/prometheus/client_model v0.4.0
	github.com/prometheus/common v0.44.0
)

require (
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
)

replace github.com/prometheus/common => github.com/beet233/common v0.44.0-compress-8

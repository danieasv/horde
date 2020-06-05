// Package apipb is the protobuf-generated stubs for the API
package apipb

//go:generate protoc -I../../../protobuf -I/usr/local/include  -I${GOPATH}/src -I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway -I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis  --go_out=plugins=grpc:. ../../../protobuf/api.proto
//go:generate protoc -I../../../protobuf -I/usr/local/include  -I${GOPATH}/src -I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway -I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis --grpc-gateway_out=logtostderr=true:. ../../../protobuf/api.proto

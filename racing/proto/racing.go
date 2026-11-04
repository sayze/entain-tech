// Package proto holds the go:generate directive that builds the racing
// gRPC code from the protos in its subdirectories.
package proto

//go:generate protoc --go_out=. --go-grpc_out=require_unimplemented_servers=false:. racing/racing.proto

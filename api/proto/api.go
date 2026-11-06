// Package proto holds the go:generate directive that builds the gateway
// code from the annotated protos in its subdirectories.
package proto

//go:generate protoc -I . --go_out . --go_opt paths=source_relative --go-grpc_out . --go-grpc_opt paths=source_relative --grpc-gateway_out . --grpc-gateway_opt paths=source_relative racing/racing.proto sports/sports.proto

#!/bin/bash
#
# Builds auto-generated protobuf files

go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

export PATH="$PATH:$(go env GOPATH)/bin"

protoc --proto_path=api-contracts/dispatcher --go_out=./internal/services/dispatcher/contracts --go_opt=paths=source_relative \
    --go-grpc_out=./internal/services/dispatcher/contracts --go-grpc_opt=paths=source_relative \
    dispatcher.proto

protoc --proto_path=api-contracts/events --go_out=./internal/services/ingestor/contracts --go_opt=paths=source_relative \
    --go-grpc_out=./internal/services/ingestor/contracts --go-grpc_opt=paths=source_relative \
    events.proto

protoc --proto_path=api-contracts/workflows --go_out=./internal/services/admin/contracts --go_opt=paths=source_relative \
    --go-grpc_out=./internal/services/admin/contracts --go-grpc_opt=paths=source_relative \
    workflows.proto

protoc --proto_path=api-contracts/workflows --go_out=./internal/services/admin/contracts/v1 --go_opt=paths=source_relative \
    --go-grpc_out=./internal/services/admin/contracts/v1 --go-grpc_opt=paths=source_relative \
    v1-admin.proto

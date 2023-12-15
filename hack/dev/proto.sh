#!/bin/bash
#
# Builds auto-generated protobuf files

protoc --proto_path=api-contracts/dispatcher --go_out=./internal/services/dispatcher/contracts --go_opt=paths=source_relative \
    --go-grpc_out=./internal/services/dispatcher/contracts --go-grpc_opt=paths=source_relative \
    dispatcher.proto

protoc --proto_path=api-contracts/events --go_out=./internal/services/ingestor/contracts --go_opt=paths=source_relative \
    --go-grpc_out=./internal/services/ingestor/contracts --go-grpc_opt=paths=source_relative \
    events.proto

protoc --proto_path=api-contracts/workflows --go_out=./internal/services/admin/contracts --go_opt=paths=source_relative \
    --go-grpc_out=./internal/services/admin/contracts --go-grpc_opt=paths=source_relative \
    workflows.proto
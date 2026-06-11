#!/bin/bash
#
# Builds auto-generated protobuf files

go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
go install connectrpc.com/connect/cmd/protoc-gen-connect-go@v1.20.0

export PATH="$PATH:$(go env GOPATH)/bin"

protoc --proto_path=api-contracts \
    --go_out=./internal/services/shared/proto/v1 \
    --go_opt=module=github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1 \
    --go-grpc_out=./internal/services/shared/proto/v1 \
    --go-grpc_opt=module=github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1 \
    v1/shared/condition.proto \
    v1/shared/trigger.proto

protoc --proto_path=api-contracts \
    --go_out=./internal/services/shared/proto/v1 \
    --go_opt=module=github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1 \
    --go-grpc_out=./internal/services/shared/proto/v1 \
    --go-grpc_opt=module=github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1 \
    v1/dispatcher.proto

protoc --proto_path=api-contracts \
    --go_out=./internal/services/shared/proto/v1 \
    --go_opt=module=github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1 \
    --go-grpc_out=./internal/services/shared/proto/v1 \
    --go-grpc_opt=module=github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1 \
    v1/workflows.proto

protoc --proto_path=api-contracts \
    --go_out=./internal/services/shared/proto/v2 \
    --go_opt=module=github.com/hatchet-dev/hatchet/internal/services/shared/proto/v2 \
    --connect-go_out=./internal/services/shared/proto/v2 \
    --connect-go_opt=module=github.com/hatchet-dev/hatchet/internal/services/shared/proto/v2 \
    --connect-go_opt=simple=true \
    v2/shared/conditions.proto \
    v2/shared/task_config.proto \
    v2/task.proto \
    v2/worker.proto \
    v2/durable_task.proto \
    v2/events.proto \
    v2/logs.proto

protoc --proto_path=api-contracts/dispatcher --go_out=./internal/services/dispatcher/contracts --go_opt=paths=source_relative \
    --go-grpc_out=./internal/services/dispatcher/contracts --go-grpc_opt=paths=source_relative \
    dispatcher.proto

protoc --proto_path=api-contracts/events --go_out=./internal/services/ingestor/contracts --go_opt=paths=source_relative \
    --go-grpc_out=./internal/services/ingestor/contracts --go-grpc_opt=paths=source_relative \
    events.proto

protoc --proto_path=api-contracts/workflows --proto_path=api-contracts \
    --go_out=./internal/services/admin/contracts --go_opt=paths=source_relative \
    --go-grpc_out=./internal/services/admin/contracts --go-grpc_opt=paths=source_relative \
    workflows.proto

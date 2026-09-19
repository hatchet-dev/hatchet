#!/bin/bash
#
# Builds auto-generated protobuf files

go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

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

# v1/operator.proto imports the package-less api-contracts/dispatcher/dispatcher.proto under its
# registered name "dispatcher.proto", so that directory is a second proto path here.
protoc --proto_path=api-contracts --proto_path=api-contracts/dispatcher \
    --go_out=./internal/services/shared/proto/v1 \
    --go_opt=module=github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1 \
    --go-grpc_out=./internal/services/shared/proto/v1 \
    --go-grpc_opt=module=github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1 \
    v1/operator.proto

# v1/serverless.proto references .AssignedAction from the same package-less file.
protoc --proto_path=api-contracts --proto_path=api-contracts/dispatcher \
    --go_out=./internal/services/shared/proto/v1 \
    --go_opt=module=github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1 \
    --go-grpc_out=./internal/services/shared/proto/v1 \
    --go-grpc_opt=module=github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1 \
    v1/serverless.proto

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

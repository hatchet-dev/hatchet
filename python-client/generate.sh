#!/bin/bash
#
# Builds python auto-generated protobuf files

poetry run python -m grpc_tools.protoc --proto_path=../api-contracts/dispatcher --python_out=./python_client --pyi_out=./python_client --grpc_python_out=./python_client dispatcher.proto
poetry run python -m grpc_tools.protoc --proto_path=../api-contracts/events --python_out=./python_client --pyi_out=./python_client --grpc_python_out=./python_client events.proto
poetry run python -m grpc_tools.protoc --proto_path=../api-contracts/workflows --python_out=./python_client --pyi_out=./python_client --grpc_python_out=./python_client workflows.proto

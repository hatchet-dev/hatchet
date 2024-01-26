#!/bin/bash
#
# Builds python auto-generated protobuf files

poetry run python -m grpc_tools.protoc --proto_path=../api-contracts/dispatcher --python_out=./hatchet_sdk --pyi_out=./hatchet_sdk --grpc_python_out=./hatchet_sdk dispatcher.proto
poetry run python -m grpc_tools.protoc --proto_path=../api-contracts/events --python_out=./hatchet_sdk --pyi_out=./hatchet_sdk --grpc_python_out=./hatchet_sdk events.proto
poetry run python -m grpc_tools.protoc --proto_path=../api-contracts/workflows --python_out=./hatchet_sdk --pyi_out=./hatchet_sdk --grpc_python_out=./hatchet_sdk workflows.proto

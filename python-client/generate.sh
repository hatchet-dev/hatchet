#!/bin/bash
#
# Builds python auto-generated protobuf files

poetry run python -m grpc_tools.protoc --proto_path=../api-contracts/dispatcher --python_out=./hatchet --pyi_out=./hatchet --grpc_python_out=./hatchet dispatcher.proto
poetry run python -m grpc_tools.protoc --proto_path=../api-contracts/events --python_out=./hatchet --pyi_out=./hatchet --grpc_python_out=./hatchet events.proto
poetry run python -m grpc_tools.protoc --proto_path=../api-contracts/workflows --python_out=./hatchet --pyi_out=./hatchet --grpc_python_out=./hatchet workflows.proto

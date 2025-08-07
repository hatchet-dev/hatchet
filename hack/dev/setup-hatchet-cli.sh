#!/usr/bin/env bash

set -euo pipefail

echo "Building frontend..."
cd ./frontend/app && npm run build
cd ../../
cp -r ./frontend/app/dist/ cmd/hatchet/cli/dist/

echo "Generating certs..."
sh ./hack/dev/generate-x509-certs.sh ./hack/dev/certs
cp -r ./hack/dev/certs/ ./cmd/internal/certs/
cp ./hack/dev/generate-x509-certs.sh ./cmd/internal/certs/generate-x509-certs.sh

echo "Building hatchet CLI..."
goreleaser build --single-target --snapshot --clean
sudo cp ./dist/hatchet_darwin_arm64_v8.0/hatchet /usr/local/bin/hatchet

#!/bin/bash

set -eux

cd frontend && (npx swagger-typescript-api -p ../bin/oas/openapi.yaml -o ./app/src/lib/api/generated -n hatchet.ts --modular --axios || cd ..)

#!/bin/bash

set -eux

set -a
. .env
set +a

cd ./frontend/app && npm run dev

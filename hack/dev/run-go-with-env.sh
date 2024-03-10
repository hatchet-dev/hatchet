#!/bin/bash

set -eux

set -a
. .env || true
set +a

exec go "$@"

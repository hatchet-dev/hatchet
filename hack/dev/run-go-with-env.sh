#!/bin/bash

set -eux

set -a
. .env
set +a

exec go "$@"

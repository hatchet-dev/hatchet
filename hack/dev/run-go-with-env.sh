#!/bin/bash

set -a
. .env
set +a

exec go "$@"

#!/bin/sh

if [ -z "$1" ]; then
  script="simple"
else
  script="$1"
fi

watchmedo auto-restart --recursive --patterns="*.py" -- poetry run $script

#!/bin/sh
set -eu

if [ -z "${CRASHDIR:-}" ]; then
    CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
fi

exec "$CRASHDIR"/start.sh stop_firewall "$@"

#!/bin/sh
set -eu

[ -z "${CRASHDIR:-}" ] && CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)

check_core() { # Go-first compatibility wrapper
    "$CRASHDIR"/start.sh check_core "$@"
}

if [ "${0##*/}" = "check_core.sh" ]; then
    check_core "$@"
fi

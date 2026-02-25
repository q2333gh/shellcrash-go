#!/bin/sh
set -eu

[ -z "${CRASHDIR:-}" ] && CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)

clash_check() { # Go-first compatibility wrapper
    "$CRASHDIR"/start.sh core_precheck
}

if [ "${0##*/}" = "clash_check.sh" ]; then
    clash_check "$@"
fi

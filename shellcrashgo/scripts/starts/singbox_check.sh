#!/bin/sh
set -eu

[ -z "${CRASHDIR:-}" ] && CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)

singbox_check() { # Go-first compatibility wrapper
    "$CRASHDIR"/start.sh core_precheck
}

if [ "${0##*/}" = "singbox_check.sh" ]; then
    singbox_check "$@"
fi

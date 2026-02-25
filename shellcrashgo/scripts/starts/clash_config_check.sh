#!/bin/sh
set -eu

[ -z "${CRASHDIR:-}" ] && CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)

check_config() { # Go-first compatibility wrapper
    "$CRASHDIR"/start.sh core_precheck "$@"
}

if [ "${0##*/}" = "clash_config_check.sh" ]; then
    check_config "$@"
fi

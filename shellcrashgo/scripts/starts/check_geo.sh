#!/bin/sh
set -eu

[ -z "${CRASHDIR:-}" ] && CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)

check_geo() { # Go-first compatibility wrapper
    [ -n "${1:-}" ] && [ -n "${2:-}" ] || return 1
    "$CRASHDIR"/start.sh check_geo "$1" "$2"
}

if [ "${0##*/}" = "check_geo.sh" ]; then
    check_geo "$@"
fi

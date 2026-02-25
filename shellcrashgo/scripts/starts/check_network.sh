#!/bin/sh
set -eu

[ -z "${CRASHDIR:-}" ] && CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)

check_network() { # Go-first compatibility wrapper
    "$CRASHDIR"/start.sh check_network
}

if [ "${0##*/}" = "check_network.sh" ]; then
    check_network "$@"
fi

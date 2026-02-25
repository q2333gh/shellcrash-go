#!/bin/sh
set -eu

[ -z "${CRASHDIR:-}" ] && CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)

core_exchange() { # upgrade selected core via Go startctl
    [ -n "${1:-}" ] || return 1
    "$CRASHDIR"/start.sh core_exchange "$1"
}

if [ "${0##*/}" = "core_exchange.sh" ]; then
    core_exchange "$@"
fi

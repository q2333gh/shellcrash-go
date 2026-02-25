#!/bin/sh
set -eu

# Go-first compatibility wrapper for legacy sourced usage:
# . "$CRASHDIR"/starts/clash_modify.sh && modify_yaml

modify_yaml() {
    [ -z "${CRASHDIR:-}" ] && CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
    "$CRASHDIR"/start.sh prepare_runtime
}

if [ "${0##*/}" = "clash_modify.sh" ]; then
    modify_yaml "$@"
fi

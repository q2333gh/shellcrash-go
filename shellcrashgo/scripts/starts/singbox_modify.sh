#!/bin/sh
set -eu

# Go-first compatibility wrapper for legacy sourced usage:
# . "$CRASHDIR"/starts/singbox_modify.sh && modify_json

modify_json() {
    [ -z "${CRASHDIR:-}" ] && CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
    "$CRASHDIR"/start.sh prepare_runtime
}

if [ "${0##*/}" = "singbox_modify.sh" ]; then
    modify_json "$@"
fi

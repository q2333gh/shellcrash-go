#!/bin/sh
# Copyright (C) Juewuy

[ -n "${__IS_MODULE_PROVIDERS_LOADED:-}" ] && return
__IS_MODULE_PROVIDERS_LOADED=1

coreconfig_run() {
    if [ -z "${CRASHDIR:-}" ]; then
        CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
    fi
    export CRASHDIR

    if command -v shellcrash-coreconfig >/dev/null 2>&1; then
        shellcrash-coreconfig --crashdir "$CRASHDIR" "$@"
        return $?
    fi

    if [ -x "$CRASHDIR/bin/shellcrash-coreconfig" ]; then
        "$CRASHDIR/bin/shellcrash-coreconfig" --crashdir "$CRASHDIR" "$@"
        return $?
    fi

    if command -v go >/dev/null 2>&1 && [ -f "$CRASHDIR/go.mod" ]; then
        go run "$CRASHDIR/cmd/coreconfig" --crashdir "$CRASHDIR" "$@"
        return $?
    fi

    echo "shellcrash-coreconfig not found and go toolchain unavailable" >&2
    return 127
}

providers() {
    coreconfig_run providers "$@"
}

if [ "${0##*/}" = "providers.sh" ]; then
    providers "$@"
fi

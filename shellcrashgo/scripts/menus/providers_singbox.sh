#!/bin/sh
# Copyright (C) Juewuy

[ -n "${__IS_PROVIDERS_SINGBOX:-}" ] && return
__IS_PROVIDERS_SINGBOX=1

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

gen_providers() {
    coreconfig_run providers-generate-singbox "$@"
}

if [ "${0##*/}" = "providers_singbox.sh" ]; then
    gen_providers "$@"
fi

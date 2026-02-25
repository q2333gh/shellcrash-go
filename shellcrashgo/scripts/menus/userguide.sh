#!/bin/sh
# Copyright (C) Juewuy

[ -n "$__IS_MODULE_USERGUIDE_LOADED" ] && return
__IS_MODULE_USERGUIDE_LOADED=1

toolsctl_run() {
    if [ -z "${CRASHDIR:-}" ]; then
        CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
    fi
    export CRASHDIR

    if command -v shellcrash-toolsctl >/dev/null 2>&1; then
        shellcrash-toolsctl --crashdir "$CRASHDIR" "$@"
        return $?
    fi

    if [ -x "$CRASHDIR/bin/shellcrash-toolsctl" ]; then
        "$CRASHDIR/bin/shellcrash-toolsctl" --crashdir "$CRASHDIR" "$@"
        return $?
    fi

    if command -v go >/dev/null 2>&1 && [ -f "$CRASHDIR/go.mod" ]; then
        go run "$CRASHDIR/cmd/toolsctl" --crashdir "$CRASHDIR" "$@"
        return $?
    fi

    echo "shellcrash-toolsctl not found and go toolchain unavailable" >&2
    return 127
}

forwhat() {
    toolsctl_run userguide "$@"
}

userguide() {
    toolsctl_run userguide "$@"
}

if [ "${0##*/}" = "userguide.sh" ]; then
    userguide "$@"
fi

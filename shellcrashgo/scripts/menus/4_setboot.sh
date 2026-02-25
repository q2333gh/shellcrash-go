#!/bin/sh
# Copyright (C) Juewuy

[ -n "${__IS_MODULE_4_SETBOOT_LOADED:-}" ] && return
__IS_MODULE_4_SETBOOT_LOADED=1

setboot_run() {
    if [ -z "${CRASHDIR:-}" ]; then
        CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
    fi
    export CRASHDIR

    if command -v shellcrash-setbootctl >/dev/null 2>&1; then
        shellcrash-setbootctl --crashdir "$CRASHDIR" "$@"
        return $?
    fi

    if [ -x "$CRASHDIR/bin/shellcrash-setbootctl" ]; then
        "$CRASHDIR/bin/shellcrash-setbootctl" --crashdir "$CRASHDIR" "$@"
        return $?
    fi

    if command -v go >/dev/null 2>&1 && [ -f "$CRASHDIR/go.mod" ]; then
        go run "$CRASHDIR/cmd/setbootctl" --crashdir "$CRASHDIR" "$@"
        return $?
    fi

    echo "shellcrash-setbootctl not found and go toolchain unavailable" >&2
    return 127
}

setboot() {
    setboot_run menu "$@"
}

if [ "${0##*/}" = "4_setboot.sh" ]; then
    setboot "$@"
fi

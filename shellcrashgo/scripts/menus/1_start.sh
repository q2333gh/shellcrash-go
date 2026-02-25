#!/bin/sh
# Copyright (C) Juewuy

[ -n "${__IS_MODULE_1_START_LOADED:-}" ] && return
__IS_MODULE_1_START_LOADED=1

# Go-first compatibility wrapper for legacy sourced usage:
# . "$CRASHDIR"/menus/1_start.sh && start_service

startover() {
    return 0
}

start_core() {
    start_service "$@"
}

start_service() {
    if [ -z "${CRASHDIR:-}" ]; then
        CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
    fi
    export CRASHDIR

    if command -v shellcrash-startctl >/dev/null 2>&1; then
        shellcrash-startctl start "$@"
        return $?
    fi

    if [ -x "$CRASHDIR/bin/shellcrash-startctl" ]; then
        "$CRASHDIR/bin/shellcrash-startctl" start "$@"
        return $?
    fi

    if [ -x "$CRASHDIR/start.sh" ]; then
        "$CRASHDIR/start.sh" start "$@"
        return $?
    fi

    if command -v go >/dev/null 2>&1 && [ -f "$CRASHDIR/go.mod" ]; then
        go run "$CRASHDIR/cmd/startctl" start "$@"
        return $?
    fi

    echo "shellcrash-startctl not found and go toolchain unavailable" >&2
    return 127
}

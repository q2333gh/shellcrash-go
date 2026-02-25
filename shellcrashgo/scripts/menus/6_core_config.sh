#!/bin/sh
# Copyright (C) Juewuy

[ -n "${__IS_MODULE_6_CORECONFIG_LOADED:-}" ] && return
__IS_MODULE_6_CORECONFIG_LOADED=1

# Go-first compatibility wrapper for legacy sourced usage:
# . "$CRASHDIR"/menus/6_core_config.sh && set_core_config

set_core_config() {
    if [ -z "${CRASHDIR:-}" ]; then
        CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
    fi
    export CRASHDIR

    if command -v shellcrash-coreconfig >/dev/null 2>&1; then
        shellcrash-coreconfig --crashdir "$CRASHDIR" menu
        return $?
    fi

    if [ -x "$CRASHDIR/bin/shellcrash-coreconfig" ]; then
        "$CRASHDIR/bin/shellcrash-coreconfig" --crashdir "$CRASHDIR" menu
        return $?
    fi

    if command -v go >/dev/null 2>&1 && [ -f "$CRASHDIR/go.mod" ]; then
        go run "$CRASHDIR/cmd/coreconfig" --crashdir "$CRASHDIR" menu
        return $?
    fi

    echo "shellcrash-coreconfig not found and go toolchain unavailable" >&2
    return 127
}

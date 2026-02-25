#!/bin/sh
# Copyright (C) Juewuy

# Go-first compatibility wrapper for legacy sourced usage:
# . "$CRASHDIR"/menus/ddns.sh && ddns_menu

ddns_menu() {
    if [ -z "${CRASHDIR:-}" ]; then
        CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
    fi
    export CRASHDIR

    if command -v shellcrash-ddnsctl >/dev/null 2>&1; then
        shellcrash-ddnsctl menu
        return $?
    fi

    if [ -x "$CRASHDIR/bin/shellcrash-ddnsctl" ]; then
        "$CRASHDIR/bin/shellcrash-ddnsctl" menu
        return $?
    fi

    if command -v go >/dev/null 2>&1 && [ -f "$CRASHDIR/go.mod" ]; then
        go run "$CRASHDIR/cmd/ddnsctl" menu
        return $?
    fi

    echo "shellcrash-ddnsctl not found and go toolchain unavailable" >&2
    return 127
}

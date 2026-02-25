#!/bin/sh
# Copyright (C) Juewuy

# Go compatibility wrapper for legacy sourced usage:
# . "$CRASHDIR"/starts/core_config.sh && get_core_config

get_core_config() {
    if [ -n "$CRASHDIR" ] && [ -f "$CRASHDIR/go.mod" ]; then
        repo_root="$CRASHDIR"
    else
        repo_root=$(pwd)
    fi

    if command -v shellcrash-coreconfig >/dev/null 2>&1; then
        shellcrash-coreconfig
        return $?
    fi

    if command -v go >/dev/null 2>&1 && [ -f "$repo_root/go.mod" ]; then
        go run "$repo_root/cmd/coreconfig"
        return $?
    fi

    echo "shellcrash-coreconfig not found and go toolchain unavailable" >&2
    return 127
}

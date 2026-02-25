#!/bin/sh
# Copyright (C) Juewuy

[ -n "${__IS_MODULE_5_TASK_LOADED:-}" ] && return
__IS_MODULE_5_TASK_LOADED=1

# Go-first compatibility wrapper for legacy sourced usage:
# . "$CRASHDIR"/menus/5_task.sh && task_menu

task_menu() {
    if [ -z "${CRASHDIR:-}" ]; then
        CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
    fi
    export CRASHDIR

    if command -v shellcrash-taskctl >/dev/null 2>&1; then
        shellcrash-taskctl --crashdir "$CRASHDIR" menu
        return $?
    fi

    if [ -x "$CRASHDIR/bin/shellcrash-taskctl" ]; then
        "$CRASHDIR/bin/shellcrash-taskctl" --crashdir "$CRASHDIR" menu
        return $?
    fi

    if command -v go >/dev/null 2>&1 && [ -f "$CRASHDIR/go.mod" ]; then
        go run "$CRASHDIR/cmd/taskctl" --crashdir "$CRASHDIR" menu
        return $?
    fi

    echo "shellcrash-taskctl not found and go toolchain unavailable" >&2
    return 127
}

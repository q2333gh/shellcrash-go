#!/bin/sh
# Copyright (C) Juewuy

# Go-first compatibility wrapper for legacy task command entry.
# Keep public function names for sourced compatibility, but route
# execution to taskctl where core logic now lives.

if [ -z "${CRASHDIR:-}" ]; then
    CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
fi
export CRASHDIR

taskctl_run() {
    if command -v shellcrash-taskctl >/dev/null 2>&1; then
        shellcrash-taskctl --crashdir "$CRASHDIR" "$@"
        return $?
    fi
    if [ -x "$CRASHDIR/bin/shellcrash-taskctl" ]; then
        "$CRASHDIR/bin/shellcrash-taskctl" --crashdir "$CRASHDIR" "$@"
        return $?
    fi
    if command -v go >/dev/null 2>&1 && [ -f "$CRASHDIR/go.mod" ]; then
        go run "$CRASHDIR/cmd/taskctl" --crashdir "$CRASHDIR" "$@"
        return $?
    fi
    echo "shellcrash-taskctl not found and go toolchain unavailable" >&2
    return 127
}

task_logger() {
    [ -n "${1:-}" ] && printf '%s\n' "$1"
}

check_update() {
    return 0
}

update_core() {
    taskctl_run update_core "$@"
}

update_scripts() {
    taskctl_run update_scripts "$@"
}

update_mmdb() {
    taskctl_run update_mmdb "$@"
}

reset_firewall() {
    taskctl_run reset_firewall "$@"
}

ntp() {
    taskctl_run ntp "$@"
}

web_save_auto() {
    taskctl_run web_save_auto "$@"
}

update_config() {
    taskctl_run update_config "$@"
}

hotupdate() {
    taskctl_run hotupdate "$@"
}

if [ $# -eq 0 ]; then
    exit 0
fi

taskctl_run "$@"

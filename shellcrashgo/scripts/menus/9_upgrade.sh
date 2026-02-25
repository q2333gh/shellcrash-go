#!/bin/sh
# Copyright (C) Juewuy

[ -n "$__IS_MODULE_9_UPGRADE_LOADED" ] && return
__IS_MODULE_9_UPGRADE_LOADED=1

upgradectl_run() {
    if command -v shellcrash-upgradectl >/dev/null 2>&1; then
        shellcrash-upgradectl --crashdir "$CRASHDIR" "$@"
        return $?
    fi

    if [ -x "$CRASHDIR/bin/shellcrash-upgradectl" ]; then
        "$CRASHDIR/bin/shellcrash-upgradectl" --crashdir "$CRASHDIR" "$@"
        return $?
    fi

    if command -v go >/dev/null 2>&1; then
        go run "$CRASHDIR/cmd/upgradectl" --crashdir "$CRASHDIR" "$@"
        return $?
    fi

    echo "shellcrash-upgradectl not found and go toolchain unavailable" >&2
    return 127
}

upgrade() {
    upgradectl_run upgrade "$@"
}

getscripts() {
    upgradectl_run setscripts "$@"
}

setscripts() {
    upgradectl_run setscripts "$@"
}

setcore() {
    upgradectl_run setcore "$@"
}

setgeo() {
    upgradectl_run setgeo "$@"
}

setdb() {
    upgradectl_run setdb "$@"
}

setcrt() {
    upgradectl_run setcrt "$@"
}

setserver() {
    upgradectl_run setserver "$@"
}

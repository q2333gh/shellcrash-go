#!/bin/sh
# Copyright (C) Juewuy

[ -n "$__IS_MODULE_OVERRIDE" ] && return
__IS_MODULE_OVERRIDE=1

coreconfig_run() {
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

override() {
    coreconfig_run override "$@"
}

setrules() {
    coreconfig_run override-rules "$@"
}

setgroups() {
    coreconfig_run override-groups "$@"
}

setproxies() {
    coreconfig_run override-proxies "$@"
}

set_clash_adv() {
    coreconfig_run override-clash-adv "$@"
}

set_singbox_adv() {
    coreconfig_run override-singbox-adv "$@"
}

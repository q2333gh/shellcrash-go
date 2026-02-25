#!/bin/sh
# Copyright (C) Juewuy

[ -n "${__IS_MODULE_2_SETTINGS_LOADED:-}" ] && return
__IS_MODULE_2_SETTINGS_LOADED=1

settingsctl_run() {
    if [ -z "${CRASHDIR:-}" ]; then
        CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
    fi
    export CRASHDIR

    if command -v shellcrash-settingsctl >/dev/null 2>&1; then
        shellcrash-settingsctl --crashdir "$CRASHDIR" "$@"
        return $?
    fi

    if [ -x "$CRASHDIR/bin/shellcrash-settingsctl" ]; then
        "$CRASHDIR/bin/shellcrash-settingsctl" --crashdir "$CRASHDIR" "$@"
        return $?
    fi

    if command -v go >/dev/null 2>&1 && [ -f "$CRASHDIR/go.mod" ]; then
        go run "$CRASHDIR/cmd/settingsctl" --crashdir "$CRASHDIR" "$@"
        return $?
    fi

    echo "shellcrash-settingsctl not found and go toolchain unavailable" >&2
    return 127
}

settings() {
    settingsctl_run menu "$@"
}

set_redir_mod() {
    settingsctl_run redir "$@"
}

set_adv_config() {
    settingsctl_run adv-ports "$@"
}

set_ipv6() {
    settingsctl_run ipv6 "$@"
}

if [ "${0##*/}" = "2_settings.sh" ]; then
    settings "$@"
fi

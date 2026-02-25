#!/bin/sh
# Copyright (C) Juewuy

[ -n "${__IS_MODULE_SUBCONVERTER_LOADED:-}" ] && return
__IS_MODULE_SUBCONVERTER_LOADED=1

coreconfig_run() {
    if [ -z "${CRASHDIR:-}" ]; then
        CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
    fi
    export CRASHDIR

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

subconverter() {
    coreconfig_run subconverter "$@"
}

gen_link_flt() {
    coreconfig_run subconverter-exclude "$@"
}

gen_link_ele() {
    coreconfig_run subconverter-include "$@"
}

gen_link_config() {
    coreconfig_run subconverter-rule "$@"
}

gen_link_server() {
    coreconfig_run subconverter-server "$@"
}

set_sub_ua() {
    coreconfig_run subconverter-ua "$@"
}

if [ "${0##*/}" = "subconverter.sh" ]; then
    subconverter "$@"
fi

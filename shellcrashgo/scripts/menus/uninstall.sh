#!/bin/sh
# Copyright (C) Juewuy

[ -n "${__IS_MODULE_UNINSTALL_LOADED:-}" ] && return
__IS_MODULE_UNINSTALL_LOADED=1

run_go_uninstall_menu() {
    if [ -z "${CRASHDIR:-}" ]; then
        CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
    fi
    export CRASHDIR

    bindir="${BINDIR:-$CRASHDIR}"
    alias_name="${my_alias:-crash}"

    if command -v shellcrash-uninstallctl >/dev/null 2>&1; then
        shellcrash-uninstallctl --crashdir "$CRASHDIR" --bindir "$bindir" --alias "$alias_name" menu
        return $?
    fi

    if [ -x "$CRASHDIR/bin/shellcrash-uninstallctl" ]; then
        "$CRASHDIR/bin/shellcrash-uninstallctl" --crashdir "$CRASHDIR" --bindir "$bindir" --alias "$alias_name" menu
        return $?
    fi

    if command -v go >/dev/null 2>&1 && [ -f "$CRASHDIR/go.mod" ]; then
        go run "$CRASHDIR/cmd/uninstallctl" --crashdir "$CRASHDIR" --bindir "$bindir" --alias "$alias_name" menu
        return $?
    fi

    echo "shellcrash-uninstallctl not found and go toolchain unavailable" >&2
    return 127
}

uninstall() {
    run_go_uninstall_menu "$@"
}

if [ "${0##*/}" = "uninstall.sh" ]; then
    uninstall "$@"
fi

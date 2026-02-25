#!/bin/sh
# Copyright (C) Juewuy

[ -n "${__IS_MODULE_SET_CRASHDIR_LOADED:-}" ] && return
__IS_MODULE_SET_CRASHDIR_LOADED=1

installpathctl_run() {
    if [ -z "${CRASHDIR:-}" ]; then
        CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
    fi
    export CRASHDIR

    if command -v shellcrash-installpathctl >/dev/null 2>&1; then
        shellcrash-installpathctl --systype "${systype:-}" "$@"
        return $?
    fi

    if [ -x "$CRASHDIR/bin/shellcrash-installpathctl" ]; then
        "$CRASHDIR/bin/shellcrash-installpathctl" --systype "${systype:-}" "$@"
        return $?
    fi

    if command -v go >/dev/null 2>&1 && [ -f "$CRASHDIR/go.mod" ]; then
        go run "$CRASHDIR/cmd/installpathctl" --systype "${systype:-}" "$@"
        return $?
    fi

    echo "shellcrash-installpathctl not found and go toolchain unavailable" >&2
    return 127
}

set_dir_from_action() {
    tmp_env=${TMPDIR:-/tmp}/shellcrash.installpath.$$.env
    if installpathctl_run --env-file "$tmp_env" "$1"; then
        # shellcheck disable=SC1090
        . "$tmp_env"
        export dir CRASHDIR
        rm -f "$tmp_env"
        return 0
    fi
    code=$?
    rm -f "$tmp_env"
    return "$code"
}

set_usb_dir() {
    set_dir_from_action usb
}

set_xiaomi_dir() {
    set_dir_from_action xiaomi
}

set_asus_usb() {
    set_dir_from_action asus-usb
}

set_asus_dir() {
    set_dir_from_action asus
}

set_cust_dir() {
    set_dir_from_action custom
}

set_crashdir() {
    set_dir_from_action select
}

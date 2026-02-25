#!/bin/sh
# Copyright (C) Juewuy

[ -n "${__IS_MODULE_DNS_LOADED:-}" ] && return
__IS_MODULE_DNS_LOADED=1

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

# Legacy sourced function compatibility.
set_dns_mod() {
    settingsctl_run dns "$@"
}

# Legacy sourced function compatibility.
fake_ip_filter() {
    settingsctl_run dns-fakeip "$@"
}

# Legacy sourced function compatibility.
set_dns_adv() {
    settingsctl_run dns-adv "$@"
}

if [ "${0##*/}" = "dns.sh" ]; then
    set_dns_mod "$@"
fi

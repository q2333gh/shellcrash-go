#!/bin/sh
set -eu

[ -z "${CRASHDIR:-}" ] && CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)

ck_cn_ipv4() { # Go-first compatibility wrapper
    "$CRASHDIR"/start.sh check_cnip ipv4
}

ck_cn_ipv6() { # Go-first compatibility wrapper
    "$CRASHDIR"/start.sh check_cnip ipv6
}

if [ "${0##*/}" = "check_cnip.sh" ]; then
    "$CRASHDIR"/start.sh check_cnip "$@"
fi

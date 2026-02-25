#!/bin/sh
# Copyright (C) Juewuy

[ -n "$__IS_MODULE_FW_FILTER_LOADED" ] && return
__IS_MODULE_FW_FILTER_LOADED=1

gatewayctl_run() {
    if [ -z "${CRASHDIR:-}" ]; then
        CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
    fi
    export CRASHDIR

    if command -v shellcrash-gatewayctl >/dev/null 2>&1; then
        shellcrash-gatewayctl --crashdir "$CRASHDIR" "$@"
        return $?
    fi

    if [ -x "$CRASHDIR/bin/shellcrash-gatewayctl" ]; then
        "$CRASHDIR/bin/shellcrash-gatewayctl" --crashdir "$CRASHDIR" "$@"
        return $?
    fi

    if command -v go >/dev/null 2>&1 && [ -f "$CRASHDIR/go.mod" ]; then
        go run "$CRASHDIR/cmd/gatewayctl" --crashdir "$CRASHDIR" "$@"
        return $?
    fi

    echo "shellcrash-gatewayctl not found and go toolchain unavailable" >&2
    return 127
}

# 流量过滤
set_fw_filter() {
    gatewayctl_run fw-filter "$@"
}

set_common_ports() {
    gatewayctl_run common-ports "$@"
}

# 自定义ipv4透明路由、保留地址网段
set_cust_host_ipv4() {
    gatewayctl_run cust-host-ipv4 "$@"
}

set_reserve_ipv4() {
    gatewayctl_run reserve-ipv4 "$@"
}

# 局域网设备过滤
fw_filter_lan() {
    gatewayctl_run lan-filter "$@"
}

if [ "${0##*/}" = "fw_filter.sh" ]; then
    set_fw_filter "$@"
fi

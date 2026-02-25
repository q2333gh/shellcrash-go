#!/bin/sh
# Copyright (C) Juewuy

[ -n "$__IS_MODULE_7_GATEWAY_LOADED" ] && return
__IS_MODULE_7_GATEWAY_LOADED=1

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

gateway() {
    gatewayctl_run menu "$@"
}

set_fw_wan() {
    gatewayctl_run fw-wan "$@"
}

set_bot_tg_config() {
    gatewayctl_run tg-config --token "$TOKEN" --chat-id "$chat_ID"
}

set_bot_tg_init() {
    gatewayctl_run tg-menu "$@"
}

set_bot_tg_service() {
    gatewayctl_run tg-service toggle "$@"
}

set_bot_tg() {
    gatewayctl_run tg-menu "$@"
}

set_vmess() {
    gatewayctl_run vmess "$@"
}

set_shadowsocks() {
    gatewayctl_run sss "$@"
}

set_tailscale() {
    gatewayctl_run tailscale "$@"
}

set_wireguard() {
    gatewayctl_run wireguard "$@"
}

if [ "${0##*/}" = "7_gateway.sh" ]; then
    gateway "$@"
fi

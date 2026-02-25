#!/bin/sh
# Copyright (C) Juewuy

[ -n "$__IS_MODULE_8_TOOLS_LOADED" ] && return
__IS_MODULE_8_TOOLS_LOADED=1

. "$CRASHDIR"/libs/logger.sh
. "$CRASHDIR"/libs/web_get_bin.sh

toolsctl_run() {
    if [ -z "${CRASHDIR:-}" ]; then
        CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
    fi
    export CRASHDIR

    if command -v shellcrash-toolsctl >/dev/null 2>&1; then
        shellcrash-toolsctl --crashdir "$CRASHDIR" "$@"
        return $?
    fi

    if [ -x "$CRASHDIR/bin/shellcrash-toolsctl" ]; then
        "$CRASHDIR/bin/shellcrash-toolsctl" --crashdir "$CRASHDIR" "$@"
        return $?
    fi

    if command -v go >/dev/null 2>&1 && [ -f "$CRASHDIR/go.mod" ]; then
        go run "$CRASHDIR/cmd/toolsctl" --crashdir "$CRASHDIR" "$@"
        return $?
    fi

    echo "shellcrash-toolsctl not found and go toolchain unavailable" >&2
    return 127
}

ddnsctl_run() {
    if [ -z "${CRASHDIR:-}" ]; then
        CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
    fi
    export CRASHDIR

    if command -v shellcrash-ddnsctl >/dev/null 2>&1; then
        shellcrash-ddnsctl --crashdir "$CRASHDIR" "$@"
        return $?
    fi

    if [ -x "$CRASHDIR/bin/shellcrash-ddnsctl" ]; then
        "$CRASHDIR/bin/shellcrash-ddnsctl" --crashdir "$CRASHDIR" "$@"
        return $?
    fi

    if command -v go >/dev/null 2>&1 && [ -f "$CRASHDIR/go.mod" ]; then
        go run "$CRASHDIR/cmd/ddnsctl" --crashdir "$CRASHDIR" "$@"
        return $?
    fi

    echo "shellcrash-ddnsctl not found and go toolchain unavailable" >&2
    return 127
}

ssh_tools() {
    toolsctl_run ssh-tools "$@"
}

# 工具与优化
tools() {
    toolsctl_run tools "$@"
}

mi_autoSSH() {
    toolsctl_run mi-auto-ssh "$@"
}

# 日志菜单
log_pusher() {
    toolsctl_run log-pusher "$@"
}
# 测试菜单
testcommand() {
    toolsctl_run testcommand "$@"
}

ddns_tools() {
    ddnsctl_run menu "$@"
}

debug() {
    toolsctl_run debug "$@"
}

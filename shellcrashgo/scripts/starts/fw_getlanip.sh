#!/bin/sh
# Copyright (C) Juewuy

getlanip() { # 获取局域网 host 地址（Go-first 兼容包装）
    [ -z "${CRASHDIR:-}" ] && CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)

    if command -v shellcrash-fwgetlanip >/dev/null 2>&1; then
        eval "$(shellcrash-fwgetlanip --crashdir "$CRASHDIR")"
        return $?
    fi

    if [ -x "$CRASHDIR/bin/shellcrash-fwgetlanip" ]; then
        eval "$("$CRASHDIR/bin/shellcrash-fwgetlanip" --crashdir "$CRASHDIR")"
        return $?
    fi

    if command -v go >/dev/null 2>&1 && [ -f "$CRASHDIR/go.mod" ]; then
        eval "$(go run "$CRASHDIR/cmd/fwgetlanip" --crashdir "$CRASHDIR")"
        return $?
    fi

    echo "shellcrash-fwgetlanip not found and go toolchain unavailable" >&2
    return 127
}

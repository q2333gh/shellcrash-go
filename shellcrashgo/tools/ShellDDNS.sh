#!/bin/sh
set -eu

if [ -z "${CRASHDIR:-}" ]; then
    CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
fi
export CRASHDIR

if command -v shellcrash-ddnsctl >/dev/null 2>&1; then
    exec shellcrash-ddnsctl menu
fi

if [ -x "$CRASHDIR/bin/shellcrash-ddnsctl" ]; then
    exec "$CRASHDIR/bin/shellcrash-ddnsctl" menu
fi

if command -v go >/dev/null 2>&1 && [ -f "$CRASHDIR/go.mod" ]; then
    exec go run "$CRASHDIR/cmd/ddnsctl" menu
fi

echo "shellcrash-ddnsctl not found and go toolchain unavailable" >&2
exit 127

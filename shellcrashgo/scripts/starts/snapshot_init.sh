#!/bin/sh
set -eu

if [ -n "${CRASHDIR:-}" ]; then
    _crashdir="$CRASHDIR"
elif command -v uci >/dev/null 2>&1; then
    _crashdir="$(uci get firewall.ShellCrash.path 2>/dev/null | sed 's/\/starts.*//')"
fi

if [ -z "${_crashdir:-}" ]; then
    _crashdir=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
fi
export CRASHDIR="$_crashdir"

if command -v shellcrash-snapshotctl >/dev/null 2>&1; then
    exec shellcrash-snapshotctl --crashdir "$CRASHDIR" "$@"
fi

if [ -x "$CRASHDIR/bin/shellcrash-snapshotctl" ]; then
    exec "$CRASHDIR/bin/shellcrash-snapshotctl" --crashdir "$CRASHDIR" "$@"
fi

if command -v go >/dev/null 2>&1 && [ -f "$CRASHDIR/go.mod" ]; then
    exec go run "$CRASHDIR/cmd/snapshotctl" --crashdir "$CRASHDIR" "$@"
fi

echo "shellcrash-snapshotctl not found and go toolchain unavailable" >&2
exit 127

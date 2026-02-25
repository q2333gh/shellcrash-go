#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)

if [ -z "${CRASHDIR:-}" ]; then
    CRASHDIR="$REPO_ROOT"
fi
export CRASHDIR

if command -v shellcrash-initctl >/dev/null 2>&1; then
    exec shellcrash-initctl "$@"
fi

if [ -x "$REPO_ROOT/bin/shellcrash-initctl" ]; then
    exec "$REPO_ROOT/bin/shellcrash-initctl" "$@"
fi

if command -v go >/dev/null 2>&1 && [ -f "$REPO_ROOT/go.mod" ]; then
    exec go run "$REPO_ROOT/cmd/initctl" --crashdir "$CRASHDIR" "$@"
fi

echo "shellcrash-initctl not found and go toolchain unavailable" >&2
exit 127

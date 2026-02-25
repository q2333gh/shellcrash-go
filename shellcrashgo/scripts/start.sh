#!/bin/sh
set -eu

# Go-only wrapper for ShellCrash lifecycle control.
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)

if command -v shellcrash-startctl >/dev/null 2>&1; then
    exec shellcrash-startctl "$@"
fi

if [ -x "$REPO_ROOT/bin/shellcrash-startctl" ]; then
    exec "$REPO_ROOT/bin/shellcrash-startctl" "$@"
fi

if command -v go >/dev/null 2>&1 && [ -f "$REPO_ROOT/go.mod" ]; then
    exec go run "$REPO_ROOT/cmd/startctl" "$@"
fi

echo "shellcrash-startctl not found and go toolchain unavailable" >&2
exit 127

#!/bin/sh
set -eu

# Go-only wrapper for minimal "meta + option-1 subconverter + hy2" flow.
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)

if command -v shellcrash-minimalflow >/dev/null 2>&1; then
    exec shellcrash-minimalflow "$@"
fi

if [ -x "$REPO_ROOT/bin/shellcrash-minimalflow" ]; then
    exec "$REPO_ROOT/bin/shellcrash-minimalflow" "$@"
fi

if command -v go >/dev/null 2>&1 && [ -f "$REPO_ROOT/go.mod" ]; then
    exec go run "$REPO_ROOT/cmd/minimalflow" "$@"
fi

echo "shellcrash-minimalflow not found and go toolchain unavailable" >&2
exit 127

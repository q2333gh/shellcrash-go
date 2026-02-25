#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)

if command -v shellcrash-menuctl >/dev/null 2>&1; then
	exec shellcrash-menuctl --crashdir "$REPO_ROOT" "$@"
fi

if [ -x "$REPO_ROOT/bin/shellcrash-menuctl" ]; then
	exec "$REPO_ROOT/bin/shellcrash-menuctl" --crashdir "$REPO_ROOT" "$@"
fi

if command -v go >/dev/null 2>&1 && [ -f "$REPO_ROOT/go.mod" ]; then
	exec go run "$REPO_ROOT/cmd/menuctl" --crashdir "$REPO_ROOT" "$@"
fi

echo "shellcrash-menuctl not found and go toolchain unavailable" >&2
exit 127

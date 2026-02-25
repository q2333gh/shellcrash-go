#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname "$0")" && pwd)

if [ -z "${CRASHDIR:-}" ]; then
	if [ -d "$SCRIPT_DIR/../configs" ]; then
		CRASHDIR="$SCRIPT_DIR/.."
	elif [ -d "$SCRIPT_DIR/../../configs" ]; then
		CRASHDIR="$SCRIPT_DIR/../.."
	else
		CRASHDIR="$SCRIPT_DIR/.."
	fi
fi
export CRASHDIR

if command -v shellcrash-taskctl >/dev/null 2>&1; then
	exec shellcrash-taskctl --crashdir "$CRASHDIR" "$@"
fi

if [ -x "$CRASHDIR/bin/shellcrash-taskctl" ]; then
	exec "$CRASHDIR/bin/shellcrash-taskctl" --crashdir "$CRASHDIR" "$@"
fi

if command -v go >/dev/null 2>&1 && [ -f "$CRASHDIR/go.mod" ]; then
	exec go run "$CRASHDIR/cmd/taskctl" --crashdir "$CRASHDIR" "$@"
fi

echo "shellcrash-taskctl not found and go toolchain unavailable" >&2
exit 127

#!/bin/sh
set -eu

if [ -z "${CRASHDIR:-}" ]; then
	CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
fi
export CRASHDIR

if command -v shellcrash-tgbot >/dev/null 2>&1; then
	exec shellcrash-tgbot --crashdir "$CRASHDIR" "$@"
fi

if [ -x "$CRASHDIR/bin/shellcrash-tgbot" ]; then
	exec "$CRASHDIR/bin/shellcrash-tgbot" --crashdir "$CRASHDIR" "$@"
fi

if command -v go >/dev/null 2>&1 && [ -f "$CRASHDIR/go.mod" ]; then
	exec go run "$CRASHDIR/cmd/tgbot" --crashdir "$CRASHDIR" "$@"
fi

echo "shellcrash-tgbot not found and go toolchain unavailable" >&2
exit 127

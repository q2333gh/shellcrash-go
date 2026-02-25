#!/bin/sh
set -eu

[ -z "${CRASHDIR:-}" ] && CRASHDIR=$(cd "$(dirname "$0")"/.. && pwd)

if command -v shellcrash-startwatchdog >/dev/null 2>&1; then
	exec shellcrash-startwatchdog --crashdir "$CRASHDIR" "$@"
fi

if [ -x "$CRASHDIR/bin/shellcrash-startwatchdog" ]; then
	exec "$CRASHDIR/bin/shellcrash-startwatchdog" --crashdir "$CRASHDIR" "$@"
fi

if command -v go >/dev/null 2>&1 && [ -f "$CRASHDIR/go.mod" ]; then
	exec go run "$CRASHDIR/cmd/startwatchdog" --crashdir "$CRASHDIR" "$@"
fi

echo "shellcrash-startwatchdog not found and go toolchain unavailable" >&2
exit 127

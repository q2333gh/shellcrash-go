#!/bin/sh
set -eu

[ -z "${CRASHDIR:-}" ] && CRASHDIR=$(cd "$(dirname "$0")"/.. && pwd)

exec "$CRASHDIR/start.sh" init "$@"

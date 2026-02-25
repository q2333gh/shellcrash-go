start_legacy() {
    [ -n "${1:-}" ] || return 1
    [ -n "${2:-}" ] || return 1

    if [ -z "${CRASHDIR:-}" ]; then
        CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
    fi

    if command -v shellcrash-legacylaunch >/dev/null 2>&1; then
        shellcrash-legacylaunch --command "$1" --name "$2"
        return $?
    fi

    if [ -x "$CRASHDIR/bin/shellcrash-legacylaunch" ]; then
        "$CRASHDIR/bin/shellcrash-legacylaunch" --command "$1" --name "$2"
        return $?
    fi

    if command -v go >/dev/null 2>&1 && [ -f "$CRASHDIR/go.mod" ]; then
        go run "$CRASHDIR/cmd/legacylaunch" --command "$1" --name "$2"
        return $?
    fi

    echo "shellcrash-legacylaunch not found and go toolchain unavailable" >&2
    return 127
}

#!/bin/sh
# Copyright (C) Juewuy

[ -n "${__IS_MODULE_BOT_TG_SERVICE_LOADED:-}" ] && return
__IS_MODULE_BOT_TG_SERVICE_LOADED=1

# Go-first compatibility wrapper for legacy sourced usage:
# . "$CRASHDIR"/menus/bot_tg_service.sh && bot_tg_start

startctl_run() {
	if [ -z "${CRASHDIR:-}" ]; then
		CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
	fi
	export CRASHDIR

	if command -v shellcrash-startctl >/dev/null 2>&1; then
		shellcrash-startctl "$@"
		return $?
	fi

	if [ -x "$CRASHDIR/bin/shellcrash-startctl" ]; then
		"$CRASHDIR/bin/shellcrash-startctl" "$@"
		return $?
	fi

	if [ -x "$CRASHDIR/start.sh" ]; then
		"$CRASHDIR/start.sh" "$@"
		return $?
	fi

	if command -v go >/dev/null 2>&1 && [ -f "$CRASHDIR/go.mod" ]; then
		go run "$CRASHDIR/cmd/startctl" "$@"
		return $?
	fi

	echo "shellcrash-startctl not found and go toolchain unavailable" >&2
	return 127
}

bot_tg_start() {
	startctl_run bot_tg_start
}

bot_tg_stop() {
	startctl_run bot_tg_stop
}

bot_tg_cron() {
	startctl_run bot_tg_cron
}

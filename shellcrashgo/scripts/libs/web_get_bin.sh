#!/bin/sh
# Copyright (C) Juewuy

[ -n "$__IS_LIB_WEB_GET_BIN" ] && return
__IS_LIB_WEB_GET_BIN=1

# Project-specific download utilities - Go-first wrapper
# Dispatches to webgetctl binary for project file downloads

webgetctl_run() {
	if [ -z "$CRASHDIR" ]; then
		CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)
	fi
	export CRASHDIR

	if command -v shellcrash-webgetctl >/dev/null 2>&1; then
		shellcrash-webgetctl "$@"
		return $?
	fi

	if [ -x "$CRASHDIR/bin/shellcrash-webgetctl" ]; then
		"$CRASHDIR/bin/shellcrash-webgetctl" "$@"
		return $?
	fi

	if command -v go >/dev/null 2>&1 && [ -f "$CRASHDIR/go.mod" ]; then
		go run "$CRASHDIR/cmd/webgetctl" "$@"
		return $?
	fi

	echo "shellcrash-webgetctl not found and go toolchain unavailable" >&2
	return 127
}

# Download project-specific files
# $1: local path, $2: remote path (relative to project root)
# $3-$6: same as webget (echo mode, redirect, cert, user agent)
get_bin() {
	local local_path="$1"
	local remote_path="$2"
	local echo_mode="$3"
	local redir_mode="$4"
	local cert_mode="$5"
	local user_agent="$6"

	# Build webgetctl command
	local cmd="webgetctl_run get-bin \"$local_path\" \"$remote_path\" --crashdir \"$CRASHDIR\""

	# Add update URL if specified
	[ -n "$update_url" ] && cmd="$cmd --update-url \"$update_url\""

	# Add URL ID if specified
	[ -n "$url_id" ] && cmd="$cmd --url-id \"$url_id\""

	# Add release type if specified
	[ -n "$release_type" ] && cmd="$cmd --release-type \"$release_type\""

	# Set proxy if CrashCore is running
	if pidof CrashCore >/dev/null 2>&1; then
		. "$CRASHDIR"/libs/set_proxy.sh
		setproxy
		[ -n "$https_proxy" ] && cmd="$cmd --proxy \"$https_proxy\""
	fi

	# Add user agent if specified
	[ -n "$user_agent" ] && cmd="$cmd --user-agent \"$user_agent\""

	# Execute Go binary
	eval "$cmd"
}

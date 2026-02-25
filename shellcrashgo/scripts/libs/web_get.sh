#!/bin/sh
# Copyright (C) Juewuy

[ -n "$__IS_LIB_WEB_GET" ] && return
__IS_LIB_WEB_GET=1

# Web download utilities - Go-first wrapper
# Dispatches to webgetctl binary for HTTP downloads

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

# Download file from URL
# $1: local path, $2: remote URL
# $3: output display (echooff), $4: redirect control (rediroff)
# $5: certificate verification (skipceroff), $6: custom user agent
webget() {
	local local_path="$1"
	local remote_url="$2"
	local echo_mode="$3"
	local redir_mode="$4"
	local cert_mode="$5"
	local user_agent="$6"

	# Build webgetctl command
	local cmd="webgetctl_run webget \"$local_path\" \"$remote_url\""

	# Set proxy if CrashCore is running
	if pidof CrashCore >/dev/null 2>&1; then
		. "$CRASHDIR"/libs/set_proxy.sh
		setproxy
		[ -n "$https_proxy" ] && cmd="$cmd --proxy \"$https_proxy\""
		cmd="$cmd --rewrite-github"
	fi

	# Add user agent if specified
	[ -n "$user_agent" ] && cmd="$cmd --user-agent \"$user_agent\""

	# Handle certificate verification
	if [ "$cert_mode" != "skipceroff" ] && [ "$skip_cert" != "OFF" ]; then
		cmd="$cmd --skip-cert"
	fi

	# Handle redirect control
	[ "$redir_mode" = "rediroff" ] && cmd="$cmd --no-redirect"

	# Execute Go binary
	eval "$cmd"
}

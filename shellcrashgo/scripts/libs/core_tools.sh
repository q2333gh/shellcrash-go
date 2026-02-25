#!/bin/sh
# Copyright (C) Juewuy

[ -n "$__IS_LIB_CORE_TOOLS" ] && return
__IS_LIB_CORE_TOOLS=1

# Core tools library - Go-first wrapper
# Dispatches to utilsctl binary for core binary management

utilsctl_run() {
	if [ -z "$CRASHDIR" ]; then
		CRASHDIR=$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)
	fi
	export CRASHDIR

	if command -v shellcrash-utilsctl >/dev/null 2>&1; then
		shellcrash-utilsctl "$@"
		return $?
	fi

	if [ -x "$CRASHDIR/bin/shellcrash-utilsctl" ]; then
		"$CRASHDIR/bin/shellcrash-utilsctl" "$@"
		return $?
	fi

	if command -v go >/dev/null 2>&1 && [ -f "$CRASHDIR/go.mod" ]; then
		go run "$CRASHDIR/cmd/utilsctl" "$@"
		return $?
	fi

	echo "shellcrash-utilsctl not found and go toolchain unavailable" >&2
	return 127
}

# $1: archive file to extract, $2: target filename
core_unzip() {
	[ -z "$TMPDIR" ] && TMPDIR="$CRASHDIR/tmp"
	[ -z "$BINDIR" ] && BINDIR="$CRASHDIR/bin"
	utilsctl_run core-unzip "$1" "$2" "$TMPDIR" "$BINDIR"
}

# Find and extract core archive from BINDIR
core_find() {
	[ -z "$TMPDIR" ] && TMPDIR="$CRASHDIR/tmp"
	[ -z "$BINDIR" ] && BINDIR="$CRASHDIR/bin"
	utilsctl_run core-find "$TMPDIR" "$BINDIR" "$CRASHDIR"
}

# $1: archive file to check
core_check() {
	[ -z "$TMPDIR" ] && TMPDIR="$CRASHDIR/tmp"
	[ -z "$BINDIR" ] && BINDIR="$CRASHDIR/bin"
	[ -z "$crashcore" ] && crashcore="clash"

	# Stop running core to prevent memory issues
	[ -n "$(pidof CrashCore)" ] && "$CRASHDIR"/start.sh stop

	# Run Go implementation
	result=$(utilsctl_run core-check "$1" "$TMPDIR" "$BINDIR" "$CRASHDIR" "$crashcore")
	ret=$?

	if [ $ret -eq 0 ]; then
		# Parse result: version=... and command=...
		core_v=$(echo "$result" | grep '^version=' | cut -d'=' -f2-)
		COMMAND=$(echo "$result" | grep '^command=' | cut -d'=' -f2-)

		# Get target and format from core type
		target_info=$(utilsctl_run core-target "$crashcore")
		target=$(echo "$target_info" | cut -d'|' -f1)
		format=$(echo "$target_info" | cut -d'|' -f2)

		# Update config files
		. "$CRASHDIR"/libs/set_config.sh
		setconfig COMMAND "$COMMAND" "$CRASHDIR"/configs/command.env
		. "$CRASHDIR"/configs/command.env
		setconfig crashcore "$crashcore"
		setconfig core_v "$core_v"
		[ -n "$custcorelink" ] && setconfig custcorelink "$custcorelink"

		export core_v COMMAND target format
	fi

	return $ret
}

# Download core from web
core_webget() {
	. "$CRASHDIR"/libs/web_get_bin.sh

	# Get core target type
	[ -z "$crashcore" ] && crashcore="clash"
	target_info=$(utilsctl_run core-target "$crashcore")
	target=$(echo "$target_info" | cut -d'|' -f1)
	format=$(echo "$target_info" | cut -d'|' -f2)

	# Determine CPU architecture
	[ -z "$cpucore" ] && {
		. "$CRASHDIR"/libs/check_cpucore.sh
	}

	[ -z "$TMPDIR" ] && TMPDIR="$CRASHDIR/tmp"

	if [ -z "$custcorelink" ]; then
		# Download from official source
		[ -z "$zip_type" ] && zip_type='tar.gz'
		get_bin "$TMPDIR/Coretmp.$zip_type" "bin/$crashcore/${target}-linux-${cpucore}.$zip_type"
	else
		# Download from custom URL
		case "$custcorelink" in
			*.tar.gz) zip_type="tar.gz" ;;
			*.gz)     zip_type="gz" ;;
			*.upx)    zip_type="upx" ;;
		esac
		[ -n "$zip_type" ] && webget "$TMPDIR/Coretmp.$zip_type" "$custcorelink"
	fi

	# Verify downloaded core
	if [ "$?" = 0 ]; then
		core_check "$TMPDIR/Coretmp.$zip_type"
	else
		rm -f "$TMPDIR/Coretmp.$zip_type"
		return 1
	fi
}

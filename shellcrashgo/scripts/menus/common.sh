# common.sh - Go-first wrapper
# Common UI functions for menus

[ -n "$__IS_COMMON_UI" ] && return
__IS_COMMON_UI=1

# Source tui_layout for tuictl_run helper
. "$CRASHDIR/scripts/menus/tui_layout.sh"

msg_alert() {
	# Default sleep time
	local _sleep_time=1

	if [ "$1" = "-t" ] && [ -n "$2" ]; then
		_sleep_time="$2"
		shift 2
	fi

	tuictl_run msg-alert "$_sleep_time" "$@"
	sleep "$_sleep_time"
}

# complete box
comp_box() {
	tuictl_run comp-box "$@"
}

top_box() {
	tuictl_run top-box "$@"
}

# bottom box
btm_box() {
	tuictl_run btm-box "$@"
}

list_box() {
	local items="$1"
	local suffix="$2"

	# Convert newline-separated items to arguments
	echo "$items" | while IFS= read -r item; do
		echo "$item"
	done | {
		local i=1
		while IFS= read -r f; do
			tuictl_run content "$i) $f$suffix"
			i=$((i + 1))
		done
	}
}

common_success() {
	tuictl_run common-success "$COMMON_SUCCESS"
}

common_failed() {
	tuictl_run common-failed "$COMMON_FAILED"
}

# =================================================
common_back() {
	tuictl_run common-back "$COMMON_BACK"
}

errornum() {
	tuictl_run error-num "$COMMON_ERR_NUM"
}

error_letter() {
	tuictl_run error-letter "$COMMON_ERR_LETTER"
}

error_input() {
	tuictl_run error-input "$COMMON_ERR_INPUT"
}

error_cancel() {
	tuictl_run error-input "$COMMON_ERR_CANCEL"
}

cancel_back() {
	tuictl_run cancel-back "$COMMON_CANCEL"
}

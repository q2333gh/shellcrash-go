# tui/layout.sh - Go-first wrapper
# Terminal UI layout helpers
# Provides menu/table formatting utilities

[ -n "$__IS_TUI_LAYOUT" ] && return
__IS_TUI_LAYOUT=1

# set the total width of the menu
TABLE_WIDTH=60

# Helper to resolve tuictl binary path
tuictl_run() {
	if command -v shellcrash-tuictl >/dev/null 2>&1; then
		shellcrash-tuictl "$@"
	elif [ -x "$CRASHDIR/bin/shellcrash-tuictl" ]; then
		"$CRASHDIR/bin/shellcrash-tuictl" "$@"
	elif [ -d "$CRASHDIR/cmd/tuictl" ]; then
		(cd "$CRASHDIR" && go run ./cmd/tuictl "$@")
	else
		echo "Error: tuictl not found" >&2
		return 1
	fi
}

# function to print content lines
content_line() {
	tuictl_run content "$@"
}

# function to print sub content lines
sub_content_line() {
	tuictl_run sub-content "$@"
}

# function to print separators
# parameter $1: pass in "=" or "-"
separator_line() {
	local separatorType="${1:-=}"
	tuictl_run separator "$separatorType"
}

# increase the spacing between forms
line_break() {
	tuictl_run line-break
}

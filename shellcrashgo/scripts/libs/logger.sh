
[ -n "$__IS_LIB_LOGGER" ] && return
__IS_LIB_LOGGER=1

#日志工具 - Go-first wrapper
#$1日志内容$2显示颜色$3是否推送$4是否覆盖上一条
logger() {
	local message="$1"
	local color="$2"
	local push="$3"
	local overwrite="$4"

	# Build loggerctl command
	local cmd="$CRASHDIR/bin/loggerctl -message \"$message\""

	# Add color if specified
	[ -n "$color" -a "$color" != "0" ] && cmd="$cmd -color \"$color\""

	# Handle push flag (default is on, disable if "off")
	[ "$push" = "off" ] && cmd="$cmd -push=false"

	# Handle overwrite flag
	[ "$overwrite" = "on" ] && cmd="$cmd -overwrite=true"

	# Add push notification config from environment
	[ -n "$device_name" ] && cmd="$cmd -device-name \"$device_name\""
	[ -n "$push_TG" ] && cmd="$cmd -push-tg \"$push_TG\" -chat-id \"$chat_ID\""
	[ -n "$push_bark" ] && cmd="$cmd -push-bark \"$push_bark\""
	[ -n "$push_Deer" ] && cmd="$cmd -push-deer \"$push_Deer\""
	[ -n "$push_Po" ] && cmd="$cmd -push-po \"$push_Po\" -push-po-key \"$push_Po_key\""
	[ -n "$push_PP" ] && cmd="$cmd -push-pp \"$push_PP\""
	[ -n "$push_Gotify" ] && cmd="$cmd -push-gotify \"$push_Gotify\""
	[ -n "$push_SynoChat" ] && cmd="$cmd -push-synochat \"$push_SynoChat\" -push-chat-url \"$push_ChatURL\" -push-chat-token \"$push_ChatTOKEN\" -push-chat-userid \"$push_ChatUSERID\""

	# Execute Go binary
	eval "$cmd"
}

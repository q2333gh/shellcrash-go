#!/bin/sh

sc_tgbot_bind_run() {
    if command -v shellcrash-tgbot >/dev/null 2>&1; then
        shellcrash-tgbot --crashdir "$CRASHDIR" "$@"
        return $?
    fi
    if [ -x "$CRASHDIR/bin/shellcrash-tgbot" ]; then
        "$CRASHDIR/bin/shellcrash-tgbot" --crashdir "$CRASHDIR" "$@"
        return $?
    fi
    if command -v go >/dev/null 2>&1 && [ -f "$CRASHDIR/go.mod" ]; then
        go run "$CRASHDIR/cmd/tgbot" --crashdir "$CRASHDIR" "$@"
        return $?
    fi
    echo "shellcrash-tgbot not found and go toolchain unavailable" >&2
    return 127
}

private_bot() {
    comp_box "请先通过 \033[32;4mhttps://t.me/BotFather\033[0m 申请TG机器人并获取其\033[36mAPI TOKEN\033[0m"
    read -r -p "请输入你获取到的API TOKEN> " TOKEN
    url_tg=https://api.telegram.org/bot${TOKEN}/getUpdates
    top_box "请向\033[32m你申请的机器人\033[33m而不是BotFather！\033[0m"
}

public_bot() {
    comp_box "请向机器人：\033[32;4mhttps://t.me/ShellCrashtg_bot\033[0m"
    TOKEN=publictoken
    url_tg=https://tgbot.jwsc.eu.org/publictoken/getUpdates
}

tg_push_token() {
    push_TG="$TOKEN"
    if ! sc_tgbot_bind_run set-push --token "$TOKEN" --chat-id "$chat_ID"; then
        return 1
    fi
    . "$CRASHDIR"/libs/logger.sh && logger "已完成Telegram日志推送设置！" 32
}

get_chatid() {
    i=1
    chat_ID=''
    while [ "$i" -le 3 ] && [ -z "$chat_ID" ]; do
        sleep 1
        comp_box "\033[33m第 $i 次尝试获取对话ID失败，正在重试...\033[0m"
        chat_ID=$(sc_tgbot_bind_run bind-chatid --url "$url_tg" --key "$public_key" 2>/dev/null || true)
        i=$((i + 1))
    done
}

set_bot() {
    public_key=$(cat /proc/sys/kernel/random/boot_id | sed 's/.*-//')
    btm_box "发送此秘钥：        \033[30;46m$public_key\033[0m"
    read -r -p "我已经发送完成(1/0)> " res
    if [ "$res" = 1 ]; then
        get_chatid
        [ -z "$chat_ID" ] && [ "$TOKEN" != 'publictoken' ] && {
            comp_box "\033[31m无法获取对话ID，请返回重新设置或手动输入ChatID！\033[0m" \
                "通常访问 \033[32;4m$url_tg\033[0m \n\033[36m即可看到ChatID\033[0m"
            read -r -p "请手动输入ChatID> " chat_ID
        }
        if echo "$chat_ID" | grep -qE '^[0-9]{8,}$'; then
            return 0
        else
            msg_alert "\033[31m无法获取对话ID，请重新配置！\033[0m"
            return 1
        fi
    fi
}

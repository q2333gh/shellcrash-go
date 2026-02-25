#!/bin/sh
# Copyright (C) Juewuy

# Go compatibility wrapper for legacy sourced usage:
# . "$CRASHDIR"/starts/fw_iptables.sh && start_iptables

fw_iptables_dispatch() {
    [ -z "${CRASHDIR:-}" ] && CRASHDIR=$(cd "$(dirname "$0")"/.. && pwd)
    "$CRASHDIR"/start.sh start_firewall "$@"
}

# Keep legacy helper names available for sourced compatibility.
start_ipt_route() {
    fw_iptables_dispatch "$@"
}

start_ipt_dns() {
    fw_iptables_dispatch "$@"
}

start_ipt_wan() {
    fw_iptables_dispatch "$@"
}

start_iptables() {
    fw_iptables_dispatch "$@"
}

#!/bin/sh
# Copyright (C) Juewuy

# Go compatibility wrapper for legacy sourced usage:
# . "$CRASHDIR"/starts/fw_nftables.sh && start_nftables

fw_nftables_dispatch() {
    [ -z "${CRASHDIR:-}" ] && CRASHDIR=$(cd "$(dirname "$0")"/.. && pwd)
    "$CRASHDIR"/start.sh start_firewall "$@"
}

# Keep legacy helper names available for sourced compatibility.
add_ip6_route() {
    fw_nftables_dispatch "$@"
}

start_nft_route() {
    fw_nftables_dispatch "$@"
}

start_nft_dns() {
    fw_nftables_dispatch "$@"
}

start_nft_wan() {
    fw_nftables_dispatch "$@"
}

start_nftables() {
    fw_nftables_dispatch "$@"
}

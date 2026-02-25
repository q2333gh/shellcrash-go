#!/usr/bin/env bash

# 简单的 Linux 端入口脚本：
# - 假设当前目录就是解压后的 shellcrashgo-linux-amd64 目录
# - 把 bin 加到 PATH
# - 设置 CRASHDIR 为当前目录
# - 直接进入 Go 菜单（menuctl）

set -e

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

export CRASHDIR="${CRASHDIR:-$ROOT_DIR}"
export PATH="$CRASHDIR/bin:$PATH"

exec "$CRASHDIR/bin/menuctl" "$@"


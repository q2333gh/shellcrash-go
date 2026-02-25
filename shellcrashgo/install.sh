#!/bin/sh
set -eu

echo "***********************************************"
echo "**                 欢迎使用                  **"
echo "**         ShellCrash Go Installer           **"
echo "**                             by  Juewuy    **"
echo "***********************************************"

echo "本安装脚本已迁移为纯 Go，实现入口为 shellcrash-installctl。"
echo
echo "请使用以下方式之一运行安装（推荐方式 1 或 2）："
echo
echo " 1) 已安装发行包："
echo "      shellcrash-installctl"
echo
echo " 2) 源码目录（本机有 Go 环境）："
echo "      cd shellcrashgo"
echo "      go run ./cmd/installctl"
echo
echo "如需自定义安装目录或其它选项，请查看："
echo "      shellcrash-installctl -h"
echo
echo "出于 pure Go 要求，当前 install.sh 不再执行任何安装逻辑。"

exit 0


#!/usr/bin/env bash

# 本地模拟 CI 中的 ShellCrash Go Linux Package 打包流程。
# 目标：
# - 为 linux/amd64 构建所有 cmd/* 入口二进制到 build/bin/
# - 打包为 dist/shellcrashgo-linux-amd64 目录
# - 生成 shellcrashgo-linux-amd64.tar.gz 供本地测试

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

echo "==> Using Go to build linux/amd64 binaries (output: build/bin)"
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0

mkdir -p build/bin
for d in ./cmd/*; do
  name="$(basename "$d")"
  echo "   - building $name"
  go build -trimpath -ldflags="-s -w" -o "build/bin/$name" "$d"
done

echo "==> Preparing dist/shellcrashgo-linux-amd64"
DIST_ROOT="dist/shellcrashgo-linux-amd64"
rm -rf dist
mkdir -p "$DIST_ROOT"

cp -a build/bin "$DIST_ROOT/"

if [ -f run_linux.sh ]; then
  cp run_linux.sh "$DIST_ROOT/"
fi

if [ -f README.md ]; then
  cp README.md "$DIST_ROOT/"
fi

echo "==> Creating shellcrashgo-linux-amd64.tar.gz"
tar -C dist -zcvf shellcrashgo-linux-amd64.tar.gz shellcrashgo-linux-amd64

echo
echo "Done."
echo "Package ready at: $ROOT_DIR/shellcrashgo-linux-amd64.tar.gz"
echo "You can test it with:"
echo "  tar -zxf shellcrashgo-linux-amd64.tar.gz"
echo "  cd shellcrashgo-linux-amd64"
echo "  ./run_linux.sh"


# ShellCrash Go 容器辅助脚本（开发者用）

本目录提供一个简单脚本，帮助开发者在本机快速把 `dist/` 下的 Linux 包放进容器里做验证。

## 一键创建容器并导入 dist

在仓库根目录执行：

```bash
cd shellcrashgo
./tests/docker/run_dist_in_container.sh
```

脚本会自动完成：

1. 检查 `dist/shellcrashgo-linux-amd64` 是否存在；如果不存在，会先运行 `./build_linux_package.sh` 生成：
   - `build/bin/*`（Go 编译好的入口二进制）
   - `dist/shellcrashgo-linux-amd64/`（用户视角的解压目录）
2. 使用 `debian:stable-slim` 创建名为 `shellcrashgo-dist-test` 的容器；
3. 将 `dist/shellcrashgo-linux-amd64/` 的内容复制到容器内的 `/app` 目录；
4. 启动容器并保持前台进程（`tail -f /dev/null`），方便你后续 `exec` 进入。

## 进入容器的 bash / sh

脚本执行完成后，你可以使用如下命令进入容器：

```bash
docker exec -it shellcrashgo-dist-test bash
# 如果镜像里没有 bash，则使用：
docker exec -it shellcrashgo-dist-test sh
```

此时容器内的 `/app` 目录就是用户视角的 ShellCrash Go Linux 包根目录，你可以在其中直接运行：

```bash
cd /app
./run_linux.sh
```

## 清理容器

测试结束后，如需删除容器：

```bash
docker rm -f shellcrashgo-dist-test
```


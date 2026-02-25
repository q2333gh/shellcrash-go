# ShellCrash Go 本地构建 / 容器构建说明（shellcrashgo）

本目录包含 **完整的 ShellCrash Go 重写代码**，以及用于构建 Docker 镜像的文件。

- 你可以在本机直接使用 Go 工具链编译所有二进制；
- 也可以选择用 `Dockerfile` 构建容器镜像。

---

## 1. 本地 Go 编译（推荐）

### 1.1 环境要求

- Go 1.22 及以上版本（见 `go.mod`）
- 标准的构建工具链（在常见 Linux 发行版/WSL/macOS 上安装官方 Go 即可）

确认 Go 已安装：

```bash
go version
```

### 1.2 一键构建所有包

在仓库根目录执行：

```bash
cd shellcrashgo
go build ./...
```

这条命令会为 `cmd/*` 下的所有入口生成可执行文件（生成在对应的 `cmd/<name>` 目录中），内部包 `internal/*` 只做编译检查。

常见入口二进制包括：

- `cmd/startctl`
- `cmd/menuctl`
- `cmd/coreconfig`
- `cmd/firewall`（如存在）
- `cmd/gatewayctl`
- `cmd/settingsctl`
- `cmd/taskctl`
- `cmd/toolsctl`
- `cmd/upgradectl`
- `cmd/tgbot`
- `cmd/tuictl`
- `cmd/utilsctl`
- 等等（完整列表可通过 `go list ./cmd/...` 查看）

如需把所有入口集中输出到统一的构建目录（推荐 `build/bin/`，便于与仓库已有 `bin/` 区分，并且可整体忽略），可以用简单脚本包装，例如：

```bash
cd shellcrashgo
mkdir -p build/bin

for d in ./cmd/*; do
  name=$(basename "$d")
  go build -o "build/bin/$name" "$d"
done
```

### 1.3 运行基础检查（可选）

> 按项目约定，测试应尽量只做“文件级 / 逻辑级验证”，不直接改动宿主机网络栈。  
> 如果你在本机运行测试，请确保接受这一点，并在有疑问时优先在隔离环境（例如容器 / 独立 namespace）中运行。

在 `shellcrashgo` 目录下：

```bash
go test ./...
```

---

## 2. CI/CD：Linux 傻瓜包与入口

CI 已提供一个专门面向 **Linux 用户端的打包 workflow**：

- workflow 名称：`ShellCrash Go Linux Package`
- 文件位置：`.github/workflows/shellcrashgo_linux_package.yaml`
- 触发方式：在 GitHub Actions 页面手动 `Run workflow`

该 workflow 会：

- 使用 Go 1.22 为 `linux/amd64` 构建所有 `cmd/*` 入口二进制（输出到 `shellcrashgo/bin`）；
- 把 `bin/`、`README.md` 和 `run_linux.sh` 打成一个 `shellcrashgo-linux-amd64.tar.gz`；
- 作为 artifact 上传，可以直接下载使用。

### 2.1 用户端使用方式（Linux）

1. 在 GitHub Actions 里运行 `ShellCrash Go Linux Package`，下载生成的 `shellcrashgo-linux-amd64.tar.gz`；
2. 在目标 Linux 机器上解压：

```bash
tar -zxf shellcrashgo-linux-amd64.tar.gz
cd shellcrashgo-linux-amd64
```

3. 直接运行傻瓜入口脚本：

```bash
chmod +x run_linux.sh
./run_linux.sh
```

- 该脚本会：
  - 把解压目录当作 `CRASHDIR`；
  - 把 `bin/` 加入 `PATH`；
  - 直接进入 Go 菜单入口（`menuctl`）。

> 目前该打包流程专注 `linux/amd64`，如需其它架构，可以在 CI 中扩展 matrix 或在本机用 `GOOS/GOARCH` 交叉编译。

---

### 2.2 本地模拟 CI 打包（开发者用）

如果你想在本地直接跑一遍与 CI 逻辑等价的打包流程（用于开发自测），可以使用仓库内置脚本：

```bash
cd shellcrashgo
./build_linux_package.sh
```

它会：

- 为 `linux/amd64` 构建所有 `cmd/*` 入口到 `build/bin/`；
- 生成 `dist/shellcrashgo-linux-amd64/` 目录；
- 在当前目录下输出 `shellcrashgo-linux-amd64.tar.gz`。

本地测试包内容与 CI 产出的结构一致，可以直接按照上面的「2.1 用户端使用方式」解压并执行 `run_linux.sh` 进行验证。

---

## 3. 使用 Docker 构建镜像

- 一台支持 Docker 的 Linux / macOS / Windows（WSL2）环境  
- 已安装：
  - Docker（推荐带 `buildx` 插件，方便多架构构建）
  - 可以访问 GitHub 或相关镜像源（Dockerfile 会在线下载内核与 s6-overlay）

建议先确认 Docker 正常工作：

```bash
docker version
docker info
```

---

## 3.1 准备构建上下文（ShellCrash.tar.gz）

`Dockerfile` 需要一个 `ShellCrash.tar.gz` 作为安装包，构建时会被复制到镜像中：

```dockerfile
COPY ShellCrash.tar.gz /tmp/ShellCrash.tar.gz
```

你需要：

1. 从 ShellCrash 项目的发布页或其它渠道获取 `ShellCrash.tar.gz` 安装包；  
2. 将该文件放到 `shellcrashgo/` 目录下，与 `Dockerfile` 位于同一目录。

最终目录结构示例：

```text
shellcrashgo/
  ├─ Dockerfile
  ├─ go.mod
  ├─ install.sh
  ├─ install_en.sh
  ├─ version
  └─ ShellCrash.tar.gz   # 需要你自己放进来
```

---

## 3.2 构建单架构 Docker 镜像

在仓库根目录或 `shellcrashgo/` 目录下执行（推荐在 `shellcrashgo/` 目录里）：

```bash
cd shellcrashgo

# 构建本地镜像，标签名可自行修改
docker build -t shellcrash:local .
```

构建成功后，使用下面命令查看镜像是否存在：

```bash
docker images | grep shellcrash
```

---

## 3.3 使用 buildx 构建多架构镜像（可选）

如果你启用了 Docker BuildKit / buildx，并希望一次构建多种架构（如 `amd64`、`arm64`、`armv7`），可以使用：

```bash
cd shellcrashgo

docker buildx build \
  --platform linux/amd64,linux/arm64,linux/arm/v7 \
  -t shellcrash:multi-arch \
  .
```

> 如需直接推送到远程仓库，将 `shellcrash:multi-arch` 替换为你自己的仓库名，例如 `docker.io/yourname/shellcrash:tag`，并追加 `--push` 选项。

---

## 3.4 运行容器（简单示例）

> 注意：运行容器时会涉及网络转发、nftables / iproute2 等操作，**请在你自己的环境中谨慎使用，不要在不熟悉的生产环境直接执行**。

最简示例（按需调整参数）：

```bash
docker run -d \
  --name shellcrash \
  --restart=unless-stopped \
  --cap-add=NET_ADMIN \
  --network host \
  shellcrash:local
```

根据你的实际网络环境，可以挂载配置目录或增加更多参数；此处仅作为“镜像可用性”的简单验证。

---

（上面本地 Go 编译流程已经涵盖了当前代码库的完整构建方式；如将来入口结构有较大调整，请同步更新本说明。）


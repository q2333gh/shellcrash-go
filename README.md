<h1 align="center">ShellCrash (Go)</h1>

<p align="center">
  <a target="_blank" href="https://github.com/MetaCubeX/mihomo/releases">
    <img src="https://img.shields.io/github/release/MetaCubeX/mihomo.svg?style=flat-square&label=Core">
  </a>
  <a target="_blank" href="https://github.com/juewuy/ShellCrash/releases">
    <img src="https://img.shields.io/github/release/juewuy/ShellCrash.svg?style=flat-square&label=ShellCrash&colorB=green">
  </a>
</p>

<p align="center">
  <strong>Go-based implementation of the ShellCrash control plane for deploying and managing mihomo / sing-box cores.</strong>
</p>

<p align="center">
  <a href="README_CN.md">简体中文</a> | English
</p>

---

## Overview

ShellCrash (Go) is the **pure Go rewrite** of the original ShellCrash project:

- All core logic is implemented in Go (start / stop, firewall, DNS, subscriptions, tasks, DDNS, init, snapshot, Telegram bot, TUI, etc.).
- Shell is kept only as a **very thin compatibility layer** for legacy entry points and init systems.
- The Go code lives entirely under the `shellcrashgo/` directory:
  - `internal/*` – domain packages (`startctl`, `firewall`, `coreconfig`, `settingsctl`, `gatewayctl`, `initctl`, `taskctl`, `toolsctl`, `snapshotctl`, `ddnsctl`, `tgbot`, `tui`, `watchdog`, etc.).
  - `cmd/*` – small CLI front-ends (one binary per controller).

If you want to understand the migration in detail, see `docs/go_rewrite_progress_summary.md`.

---

## Architecture (Go-first)

- **Core responsibilities (Go only):**
  - Lifecycle: `startctl`, `watchdog`, `initctl`, `setbootctl`.
  - Network & firewall: `firewall`, `gatewayctl`.
  - Config pipeline: `coreconfig` (Clash / sing-box providers, overrides, minimal-flow).
  - Runtime menus / TUI: `menuctl`, `tui`, `tuictl`.
  - Tools & automation: `toolsctl`, `taskctl`, `ddnsctl`, `upgradectl`, `utilsctl`.
  - Integrations: `tgbot`, snapshot / firmware helpers, logger.
- **Shell layer (compatibility only):**
  - Small wrappers under `shellcrashgo/scripts` and `shellcrashgo/tools` which:
    - Preserve historical file/function names for old firmware hooks.
    - Locate the corresponding Go binary (e.g. `shellcrash-startctl`, `shellcrash-menuctl`) and forward arguments.
  - Heavy shell installers (`install.sh`, `install_en.sh`) have been retired; they no longer contain installation logic.

From a runtime perspective, this repository is a **Go application with optional shell shims**, not a shell script project.

---

## Installation (Go-based)

> **Note**  
> The preferred way to install and manage ShellCrash (Go) is via the Go CLI binaries.  
> The old `curl | sh install.sh` and `install_en.sh` flows are no longer used for real work.

### 1. Build from source (Linux)

Requirements:

- Go 1.22+ (recommended).
- A recent Linux distribution (or OpenWrt toolchain rootfs) with standard build tools.

Steps:

```sh
cd shellcrashgo

# Run tests (recommended)
go test ./...

# Option A: build all cmd/* binaries into build/bin
./build_linux_package.sh

# Optionally extract the test package
tar -zxf shellcrashgo-linux-amd64.tar.gz
cd shellcrashgo-linux-amd64

# Start the TUI menu directly from Go binaries
./bin/menuctl --crashdir "$(pwd)"
```

You can also build individual controllers:

```sh
cd shellcrashgo
go build -o bin/menuctl ./cmd/menuctl
go build -o bin/startctl ./cmd/startctl
```

### 2. Go installer entrypoint

For scripted installs or packaging, use the pure Go installer:

```sh
cd shellcrashgo
go run ./cmd/installctl \
  --crashdir /etc/ShellCrash \
  --tmpdir   /tmp/ShellCrash \
  --url      "https://testingcf.jsdelivr.net/gh/juewuy/ShellCrash@master"
```

`installctl` will:

- Download the `version` and `ShellCrash.tar.gz` assets from the given base URL.
- Extract into `--crashdir`.
- Run `initctl` to create/normalize `ShellCrash.cfg`, `command.env`, basic directories, and init hooks.

The legacy `install.sh` / `install_en.sh` files remain only as **informational stubs** that print how to run `installctl`; they do not perform installation themselves.

---

## Usage

Once installed into a `CRASHDIR` (for example `/etc/ShellCrash`), typical entrypoints are:

```sh
# Main TUI / menu
shellcrash-menuctl --crashdir "$CRASHDIR"

# Or, if PATH includes the bin directory produced by packaging:
menuctl --crashdir "$CRASHDIR"

# Start / stop / restart core services
startctl start   --crashdir "$CRASHDIR"
startctl stop    --crashdir "$CRASHDIR"
startctl restart --crashdir "$CRASHDIR"
```

Legacy aliases such as `crash` can still be configured by the init logic to point to the Go menu binaries, depending on your environment and firmware.

---

## Development

This repository is optimized for development and automated testing:

- **Tests**
  - Run all tests from the Go rewrite:
    ```sh
    cd shellcrashgo
    go test ./...
    ```
- **Code layout**
  - Go code: `shellcrashgo/internal/...`, `shellcrashgo/cmd/...`
  - Thin shell wrappers (compat): `shellcrashgo/scripts`, `shellcrashgo/tools`
  - Docs about the migration: `docs/go_rewrite_progress_summary.md`, `docs/arch.md`

The guiding principles are:

- Keep **all behavior and side effects** in Go.
- Keep shell scripts **minimal, readable, and removable**.
- Prefer small, well-tested internal packages over large monolithic modules.

---

## Links

- Original ShellCrash project: `https://github.com/juewuy/ShellCrash`
- FAQ & docs: `https://juewuy.github.io/`
- Telegram discussion: `https://t.me/ShellClash`

---

## License

This project is licensed under the [GNU General Public License v3.0](LICENSE.txt).

<h1 align="center">ShellCrash</h1>

<p align="center">
  <a target="_blank" href="https://github.com/MetaCubeX/mihomo/releases">
    <img src="https://img.shields.io/github/release/MetaCubeX/mihomo.svg?style=flat-square&label=Core">
  </a>
  <a target="_blank" href="https://github.com/juewuy/ShellCrash/releases">
    <img src="https://img.shields.io/github/release/juewuy/ShellCrash.svg?style=flat-square&label=ShellCrash&colorB=green">
  </a>
</p>

<p align="center">
  <strong>A powerful script tool for the convenient deployment and management of mihomo/sing-box kernels in Shell environments.</strong>
</p>

<p align="center">
  <a href="README_CN.md">简体中文</a> | English
</p>

---

## :rocket: Core Features

- **Multi-Kernel Support**: Easily manage and switch between **mihomo** and **sing-box** kernels directly within the Shell environment.
- **Flexible Configuration Management**: Supports online import of subscription links and configuration files to simplify the setup process.
- **Automated Tasks**: Configure scheduled tasks for automatic updates of configuration files and rules.
- **Graphical Dashboard**: Support for online installation and use of local Web Dashboards to intuitively manage built-in rules and traffic.
- **Multiple Operation Modes**: Supports switching between various traffic forwarding modes, including Router mode and Local mode.
- **One-Click Maintenance**: Built-in online update functionality to keep the script and features up to date.

## :computer: Device Support

ShellCrash is designed to be compatible with the vast majority of network devices based on the Linux kernel:

* **Router Devices**: Supports various firmwares based on OpenWrt or its derivatives (e.g., Xiaomi, Netgear etc.).
* **Linux Servers**: Supports devices running standard Linux/GNU distributions (e.g., Debian, CentOS, Armbian, Ubuntu, etc.).
* **Third-Party Firmware**: Compatible with Padavan (Conservative Mode), Pandora, and ASUS/Merlin firmware.
* **Other Devices**: Compatible with other devices based on Linux/GNU or Linux/busybox.
* **Docker**：Compatible with Docker environments (e.g., Synology, PVE, etc.).

> For additional device support, please submit an [Issue](https://github.com/juewuy/ShellCrash/issues) or provide feedback in the [Telegram Group](https://t.me/ShellClash) (please include the device model and the output of the `uname -a` command).

---

## :hammer_and_wrench: Installation Guide

> [!TIP]
> If you encounter connection failures or SSL-related issues, please try switching to an alternative installation mirror.

### Prerequisites
1. Ensure the device has **SSH** enabled and **Root privileges** obtained (Linux systems with a GUI can use the terminal directly).
2. Connect to the device using an SSH tool (such as PuTTY, JuiceSSH, or the system's built-in terminal).

### :penguin: Standard Linux Device Installation

> [!IMPORTANT]
> Please perform the installation as the root user.

> Install via wget (jsDelivr CDN source)
```sh
export url='https://testingcf.jsdelivr.net/gh/juewuy/ShellCrash@dev' \
  && wget -q --no-check-certificate -O /tmp/install.sh $url/install_en.sh \
  && bash /tmp/install.sh \
  && . /etc/profile &> /dev/null
```

> Or install via curl (Author's private source)

```sh
export url='https://gh.jwsc.eu.org/dev' && bash -c "$(curl -kfsSl $url/install_en.sh)" && . /etc/profile &> /dev/null
```

### :satellite: Router Device Installation

**Installation via `curl`:**
> GitHub Source (Recommended for overseas environments or environments with proxy access)
```sh
export url='https://raw.githubusercontent.com/juewuy/ShellCrash/dev' \
  && sh -c "$(curl -kfsSl $url/install_en.sh)" \
  && . /etc/profile &> /dev/null
```

> Or jsDelivr CDN source

```sh
export url='https://testingcf.jsdelivr.net/gh/juewuy/ShellCrash@dev' \
  && sh -c "$(curl -kfsSl $url/install_en.sh)" \
  && . /etc/profile &> /dev/null
```

> Or Author's private source
```sh
export url='https://gh.jwsc.eu.org/dev' && sh -c "$(curl -kfsSl $url/install_en.sh)" && . /etc/profile &> /dev/null
```

**Installation via `wget`:**
> GitHub Source (Recommended for overseas environments or environments with proxy access)
```sh
export url='https://raw.githubusercontent.com/juewuy/ShellCrash/dev' \
  && wget -q --no-check-certificate -O /tmp/install.sh $url/install_en.sh \
  && sh /tmp/install.sh \
  && . /etc/profile &> /dev/null
```

> Or jsDelivr CDN source
```sh
export url='https://testingcf.jsdelivr.net/gh/juewuy/ShellCrash@dev' \
  && wget -q --no-check-certificate -O /tmp/install.sh $url/install_en.sh \
  && sh /tmp/install.sh \
  && . /etc/profile &> /dev/null
```

### :pager: Installation for Legacy Devices with Older `wget` Versions

> Author's private HTTP beta source
```sh
export url='http://t.jwsc.eu.org' \
  && wget -q -O /tmp/install.sh $url/install_en.sh \
  && sh /tmp/install.sh \
  && . /etc/profile &> /dev/null
```


### :cloud: Virtual Machines
- **Alpine Linux VM**: It is highly recommended to use an Alpine image for optimal compatibility.
```sh
# Install necessary dependencies
apk add --no-cache wget openrc ca-certificates tzdata nftables iproute2 dcron
# Execute installation command
export url='https://testingcf.jsdelivr.net/gh/juewuy/ShellCrash@dev' \
  && wget -q --no-check-certificate -O /tmp/install.sh $url/install_en.sh \
  && sh /tmp/install.sh \
  && . /etc/profile &> /dev/null
```

 ### :whale: Docker 

 Please visit the official Docker image:

- [ShellCrash on Docker Hub](https://hub.docker.com/r/juewuy/shellcrash)


### :package: Local Installation

If online installation is not possible, please follow the guide for local installation:

- [Local ShellCrash Installation Tutorial | Juewuy's Blog](https://juewuy.github.io/bdaz)

---

## :book: Usage Instructions

After installation, enter the following commands in the terminal to launch the management interface:

```shell
crash        # Launch the interactive script menu
crash -h     # View the list of command help
```

### Running Dependencies
| Component | Necessity | Description |
| :--- | :--- | :--- |
| curl / wget | Mandatory | Required for node saving, online installation, and update operations. |
| iptables / nftables | Critical | Without these, the script can only run in Pure Mode. |
| crontab | Low | Required for scheduled tasks; otherwise, they will not function. |
| net-tools | Very Low | Used for automatic port occupancy detection. |
| ubus / iproute-doc | Very Low | Used for automatically obtaining the local Host address. |

---

## :link: Related Links
- FAQ: [Juewuy's Blog](https://juewuy.github.io/chang-jian-wen-ti/)
- Changelog: [Release History](https://github.com/juewuy/ShellCrash/releases)
- Discussion: [Telegram Group](https://t.me/ShellClash)

## :scroll: License

This project is licensed under the [GNU General Public License v3.0](LICENSE.txt).

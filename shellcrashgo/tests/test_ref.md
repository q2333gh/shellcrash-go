### 这类网络程序一般怎么做“大型完整测试”

先说结论：**不会直接在开发机的真实网络环境上乱改，而是分层测试 + 隔离环境**。在你这个 `ShellCrash-dev` 目录里，其实已经能看到这套思路的雏形。

---

### 1. 分层测试：先把“会动网络的那一层”隔离开

**第一层：纯逻辑/配置流水线测试（不碰内核网络）**

你现在看到的这个测试就是典型代表：

```12:59:/home/jwk/code/my_openwrt/ShellCrash-dev/ShellCrash-dev/tests/test_minimal_flow_default_meta_hy2_subconvert_py.py
with tempfile.TemporaryDirectory() as td:
    ...
    crashdir = tdp / "ShellCrash"
    (crashdir / "configs").mkdir(parents=True)
    (crashdir / "yamls").mkdir(parents=True)
    (crashdir / "jsons").mkdir(parents=True)
    ...
    env["CRASHDIR"] = str(crashdir)
    ...
    proc = subprocess.run(
        ["python3", str(script)],
        cwd=str(repo_root),
        env=env,
        capture_output=True,
        text=True,
    )

    assert proc.returncode == 0, proc.stderr + proc.stdout
    cfg_text = (crashdir / "configs" / "ShellCrash.cfg").read_text(encoding="utf-8")
    out_yaml = (crashdir / "yamls" / "config.yaml").read_text(encoding="utf-8")

    assert "crashcore=meta" in cfg_text
    assert f"Https='file://{tdp}/sub?" in cfg_text
    assert "type: hysteria2" in out_yaml
```

- **特点**：
  - 用 `tempfile.TemporaryDirectory()` 搞一个临时 `CRASHDIR`，完全不碰真实 `/etc/ShellCrash`。
  - 自己写入 `providers.cfg / providers_uri.cfg / servers.list / sub`，模拟你“默认 Mihomo + Subconvert 选项1 + HY2 节点”的场景。
  - 跑一遍 `minimal_flow_default_meta_hy2_subconvert.py`，然后只检查生成的 `ShellCrash.cfg` 和 `config.yaml` 内容。
- **效果**：把“下载订阅 → 组装 URL → 生成 yaml/json”这条链路完整走一遍，但**完全不牵扯 iptables/nftables、路由表、内核转发**，对开发机网络是 0 风险。

像这样你可以扩展出一整套“大覆盖率”的测试用例：

- 不同 `crashcore`、不同 `providers` 组合、
- 不同 `server_link/rule_link`、
- 不同 UA / include / exclude …

这一层就能把 **80% 以上的逻辑 bug** 在“不动网络”的前提下干掉。

---

### 2. 真正“会改网络”的 E2E 测试：放进隔离环境

等到要测“真流量从 A 出去、经 ShellCrash 转发到 B”的时候，才需要碰 iptables / nftables / 路由转发，这一层一般有三种做法，**都不直接糊在开发机主命名空间上**：

#### **方案 A：用 Docker 容器做测试沙箱（最常见）**

你这里已经有一个专门的 Dockerfile：

```49:80:/home/jwk/code/my_openwrt/ShellCrash-dev/ShellCrash-dev/Dockerfile
FROM alpine:latest
...
RUN apk add --no-cache \
    wget \
    ca-certificates \
    tzdata \
    nftables \
    iproute2
...
COPY --from=builder /etc/ShellCrash /etc/ShellCrash
COPY --from=builder /tmp/CrashCore.tar.gz /etc/ShellCrash/CrashCore.tar.gz
COPY --from=builder /usr/bin/crash /usr/bin/crash
...
ENTRYPOINT ["/init"]
```

典型用法就是：

- **容器内的 iptables/nftables、路由表、转发规则** 全部关在容器自己的 network namespace 里。
- 开发机只是一个 Docker host，本机的网络栈几乎不受影响。
- 你可以用 `docker run --cap-add=NET_ADMIN` 或 `--privileged` 给容器权限，在容器内随便折腾规则。
- 再用 `docker network` / `docker compose` 起一些“假上游服务”（HTTP 服务、测速服务等），发包走一圈验证。

这样你就可以做“比较完整的端到端测试”，但影响范围只在容器内部。

#### **方案 B：Linux `ip netns` 创建独立网络命名空间**

如果不想用 Docker，也可以直接用内核原生的 netns：

- `ip netns add testns`
- 建一个 veth pair，把一端放进 `testns`，另一端留在 host。
- 在 `testns` 里起 ShellCrash，它只会改 `testns` 的 iptables/nftables。
- host 上只看到一条到 veth 的路由，对主网络配置影响极小。

本质跟 Docker 是一样的：**换一个 network namespace，当成独立“路由器小世界”来测。**

#### **方案 C：直接在 R2S/路由器设备上做自动化回归**

结合你现在要解决的“R2S 上 ShellCrash 不能用”问题，**最保险的“全链路测试”环境其实就是：**

- 在一台专门用于测试的 R2S 上部署 ShellCrash（不是你日常上网的主路由）。
- 上游接一个可控的上级（比如你的主路由），下游接一台测试 PC。
- 用脚本/CI 通过 SSH：
  - 下发配置（比如把 `ShellCrash-dev` 里验证过的 providers/servers 组合同步过去），
  - 重启 ShellCrash，
  - 然后从测试 PC 做一套连通性检查（ping、curl、域名解析、特定站点访问）。
- 这样真正动路由/转发的只有这台测试 R2S，你的开发机网络完全不受影响。

---

### 3. 回答你关心的点

- **“这种网络程序如何做大型完整测试？”**
  - **分两层**：  
    - **配置与逻辑层**：像现在这个 `test_minimal_flow_default_meta_hy2_subconvert_py.py`，用临时目录、环境变量，把 Mihomo + Subconvert + HY2 这条链路的所有“文本/配置逻辑”走通。可以写很多 case，覆盖度很高，而且**完全不动网络**。  
    - **真实流量层**：放进 **Docker 容器 / netns / 专用 R2S 测试机** 里完成，对宿主开发机只是“外面绕了一个网关”，不会大改你自己的路由/iptables。
- **“不会影响开发机器的网络吗？”**
  - 只要你遵守这两个原则：  
    - **在本机只跑“不动 iptables 的测试”（配置流水线）**；  
    - **所有会改 iptables/nftables/路由的 ShellCrash 启动，都放进容器 / netns / 物理测试路由器**；  
  - 那开发机的网络就基本是安全的，最多就是 Docker bridge 上多几个容器。

---

如果你愿意，下一步我可以帮你：  
- 把现在这个 `minimal_flow_default_meta_hy2_subconvert` 的测试思路，扩展成一组“覆盖你 R2S 实际用法”的用例清单；  
- 或者设计一个最小的 Docker 测试拓扑，用来专门复现/调试“R2S 上可用性”和你当初“烂路由器默认可用”的差异。
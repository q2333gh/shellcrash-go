## ShellCrash-dev 开发约定（给 AI / 测试用）

**核心原则：在这台开发机上跑测试时，不要触碰真实网络栈。**

- **在本机只跑“纯配置 / 纯逻辑”测试**  
  - 可以参考 `tests/test_minimal_flow_default_meta_hy2_subconvert_py.py` 这种写法：  
    - 使用 `tempfile.TemporaryDirectory()` 或等价方式搭建临时 `CRASHDIR`。  
    - 只读写临时目录下的 `configs/、yamls/、jsons/` 等文件。  
    - 通过断言生成的配置内容（如 `ShellCrash.cfg`、`config.yaml`）来验证行为。  
  - 这类测试**不得**修改宿主机的 iptables、nftables、路由表或内核转发设置。

- **任何会改 iptables/nftables/路由的操作，一律不要在本机直接执行**  
  - 如果必须做“真流量 / 真路由”测试，请：  
    - 放到 Docker 容器、独立 network namespace，或专用测试路由器 / R2S 上执行；  
    - 在这些隔离环境内随意修改网络规则，不要在本机默认 network namespace 里做实验。  
    - AI / Agent 如需自测，只能使用**短生命周期的受限 Docker 容器**（默认 bridge 网络），严禁使用 `--net=host`、`--privileged`、`--cap-add=NET_ADMIN` 等会暴露宿主网络栈的参数。

- **禁止事项（本机开发机环境）**  
  - 不要在本机直接运行会持久修改网络栈的 ShellCrash 启动脚本或 init 脚本。  
  - 不要在测试流程中尝试更改默认网关、DNS、系统代理或全局流量转发。  
  - 如有疑问，一律选择“只动文件、不动网络”的测试方式。

简而言之：在 `ShellCrash-dev` 仓库内，测试默认应当是**文件级 / 逻辑级验证**，真正动网络的实验必须迁移到隔离环境，不要影响核心开发机网络。


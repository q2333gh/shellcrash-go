# TUI / CLI 设计资料整理

> 本文是若干 TUI / CLI 设计最佳实践文章的摘录与要点整理，方便本地查阅与后续扩展为 agent skill。

## 1. 核心设计哲学 / 文章

### 《The TUI Commandments》

- 原文链接：<https://bczsalba.com/post/the-tui-commandments>
- 重点思想：
  - 终端是自己的 UI 范式，不要简单照搬 GUI / Web。
  - 输入可预测、输出稳定：同样的操作永远产生同样的布局和键位行为。
  - 以按键为单位设计交互（而不是一整条命令），但同时要确保可脚本化。
  - 启动 / 退出要极快，状态持久化而不是长时间常驻。
  - 可配置性：快捷键、颜色、布局都要给用户留钩子。
  - 这篇更偏方法论，非常适合作为 TUI CLI 的“世界观”。

### 《How to Build Beautiful Terminal User Interfaces in Python》（TNG 博客）

- 原文链接：<https://blog.tng.sh/2025/10/how-to-build-beautiful-terminal-user.html>
- 重点思想：
  - 强调集中管理主题 / 样式（Theme）：颜色、图标、布局都放在一个 `Theme` 类里。
  - 提倡做一个 `BaseUI` 基类，所有 screen / page 继承，统一：布局骨架、常用组件（标题栏、状态栏等）。
  - 把“业务逻辑”和“渲染逻辑”拆开，方便后期扩展 / 重构。

## 2. 可当最佳实践阅读的框架文档

### Rust `tui-rs` 文档

- 原文链接：<https://docs.rs/tui/latest/tui/>
- 可借鉴的模式：
  - 基于「组件 + 布局树」的思路（`Block` / `Layout` / `Widget`）。
  - 事件循环、渲染循环的结构化：`draw -> handle input -> update state`。
  - 适合作为 Go / Rust 等语言自建事件循环和组件抽象的参考。

### Textual & Rich（Python）

- Textual 教程：<https://realpython.com/python-textual>
- CLI & TUI 视频：<https://realpython.com/videos/developing-clis-tuis>
- 值得学习的点：
  - 消息 / 事件驱动模型（Message / Event Bus）。
  - 组件化 + 状态驱动渲染（有点像 React 但跑在 Terminal）。
  - 如何做“有层级的界面 + 路由（screen 切换）”。

## 3. 直接可用的设计建议（与代码结构直接相关）

### 架构层次

- **Core**：纯业务逻辑（配置、执行命令、状态机）。
- **UI**：只关心“现在 state 怎么渲染到 terminal 上”。
- **IO / 适配层**：键盘事件、终端大小变化等，统一转换成内部事件。

### 事件循环模式

- 主循环统一为：
  - 读取输入事件（按键、resize、定时器）。
  - 更新内部 state。
  - 根据 state 重绘整个 UI（或至少一个「区域」）。

### 状态设计

- 集中管理一个 `AppState`：
  - 当前 screen / mode（如：列表、详情、编辑）。
  - 选中项索引。
  - 滚动偏移。
  - 全局提示 / 错误信息。
- 所有 UI 只读 `AppState`，修改通过 action / command 完成。

### 键盘交互规范

- 使用通用习惯：
  - `h/j/k/l` 或方向键移动。
  - `q` 退出，`?` 打开帮助，`/` 搜索。
  - `Esc` 回到上一级 / 取消操作。
- 所有快捷键应在界面某处可见（底部 status bar 或 `?` 的 help 面板）。

### 可配置性

- 颜色和 keybinding 用配置文件或环境变量覆盖。
- 日志输出可选地写到文件，而不是直接污染 UI。


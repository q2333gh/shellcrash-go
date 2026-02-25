---
name: tui-cli-design
description: Design and review terminal user interfaces (TUIs) and interactive CLIs using proven patterns for architecture, state, keybindings, accessibility, and UX. Use when building or refactoring TUI apps, curses-style dashboards, or when the user asks about terminal UI / CLI UX design.
---

# TUI & CLI Design Best Practices

This skill guides the agent when designing or refactoring **terminal user interfaces (TUIs)** and **interactive CLIs**. It distills patterns from:

- `docs/tui/reading-notes.md`
- `docs/tui/the-tui-commandments.md`
- `docs/tui/beautiful-terminal-uis-python.md`
- `docs/tui/inclusive-cli-best-practices.md`
- `docs/tui/julia-evans-terminal-rules.md`

Keep TUIs **predictable, fast, accessible, and script‑friendly**.

---

## 1. Decide: CLI or TUI?

Before suggesting UI changes, classify the task:

- **Prefer plain CLI (non-interactive) when**:
  - The job is essentially a **single-step operation** (`tool subcmd args`).
  - Output is naturally piped, logged, or parsed by scripts.
  - Interactivity would just add friction.
- **Reach for TUI when**:
  - The user needs to **explore** data, tweak options, or perform multi-step workflows.
  - Layout, selection, or navigation benefits from a visual structure.

When in doubt:

- Start from a **solid CLI** and add TUI layers **only where complexity demands it**.

When designing a TUI for an existing CLI:

- Ensure **all essential operations remain possible via CLI flags** (for automation).

---

## 2. Core Philosophy (TUI “Commandments”)

When working on a TUI, bias design towards:

1. **Do few things exceptionally well**
   - Keep the app focused on clearly defined tasks.
   - Avoid feature bloat that harms clarity or speed.

2. **Deterministic interface**
   - Same initial state + same inputs ⇒ same layout and results.
   - Separate:
     - **Inputs** (controls / keybindings / menus).
     - **Content** (dynamic data).
   - Keep input positions and labels stable across refreshes.

3. **Keypress-first interaction**
   - Assume a hardware keyboard by default.
   - Every action must be reachable via keybindings.
   - Mouse support is optional and secondary.

4. **User-configurable ergonomics**
   - Keybindings and, ideally, theme/layout are configurable via files/env.
   - Store bindings and theme in **human-editable** config.

5. **Use CLI arguments where possible**
   - Flags can:
     - Pre-select views or filters.
     - Inject data through stdin.
     - Provide one-off options.

6. **Launch and exit quickly**
   - Optimize for **time-to-task-completion**:
     - `shell prompt → perform task → back to prompt`.
   - Avoid “fancy” loading screens and heavy startup work.

7. **Do not reimplement terminal behaviour**
   - Let the terminal handle:
     - Drag-to-select for copy.
     - Native scrollback when possible.
   - Avoid fighting multiplexers (tmux, screen) or emulator features.

8. **Use TUI with purpose**
   - Do not wrap simple one-shot commands in a TUI.
   - Reserve TUI complexity for **multi-step or exploratory** workflows.

---

## 3. Architecture & State

When proposing implementation details, structure the app into three layers:

1. **Core (domain / business logic)**
   - Config parsing and validation.
   - State machine / operations (e.g., run tests, show logs, manage jobs).
   - Pure functions whenever possible (easy to test).

2. **UI (rendering)**
   - Given an `AppState`, decides **what appears where** on screen.
   - No direct system calls besides drawing.

3. **IO / Adapter layer**
   - Reads key events, resize events, timers.
   - Translates them into **internal actions**.
   - Orchestrates the **event loop**:
     - Read events → update state → redraw.

### AppState design

When changing or creating state, ensure it explicitly tracks:

- Current **screen / mode** (list, detail, edit, help, etc.).
- Selection indices (which item is active).
- Scroll offsets (per list / pane if needed).
- Global status:
  - Info / warning / error messages.
  - In-progress operations.

Rules:

- UI code **reads** `AppState` only.
- Mutations happen via **named actions / commands**, not ad-hoc field changes.

---

## 4. Keybindings & Behaviour Conventions

Default expectations from typical terminal users:

- **Quit / exit**
  - `q` to quit TUIs (unless role is more like a window manager or editor).
  - `Ctrl-C`:
    - Non-interactive tools: quit immediately (default SIGINT behaviour).
    - Interactive programs: cancel current operation, not necessarily exit.
  - REPLs: `Ctrl-D` on an empty line should quit.

- **Movement**
  - `↑/↓/←/→` and/or `h/j/k/l` for navigation.
  - `PgUp/PgDn` or `Ctrl-U/Ctrl-D` for paging.

- **Search & help**
  - `/` to search (with `n` / `N` to navigate results where appropriate).
  - `?` to show help / keybindings overview.

### Implementation guidance

When designing or modifying controls:

- Stick to these conventions **unless there is a strong reason not to**, and document any deviations in the help view.
- Provide a focused **help panel / screen** that:
  - Lists primary actions and keybindings.
  - Is reachable by a single keypress (`?`).

---

## 5. Visual Design & Theming

Use a **centralized theme system**:

- A single theme module / struct / class manages:
  - Semantic colours (e.g., `primary`, `success`, `warning`, `muted`).
  - Icons (Unicode / emojis if appropriate).
  - Layout constants (padding, box styles, spacing).
  - Text styles (title, header, dim notes, error).

When suggesting changes:

- Prefer **semantic style references** in code (e.g., `theme.Colors.PRIMARY`) over hardcoded colour strings.
- Encourage shared helpers:
  - Base UI component / base screen with common patterns (centered panels, tables, status bars).

Colour usage:

- Prefer **base ANSI colours** (16-color palette) for compatibility.
- Avoid assuming specific background colours.

---

## 6. Accessibility & Inclusive CLI Behaviour

When adjusting output / rendering or suggesting options:

- **Respect non-TTY output**
  - Disable colour and decorative formatting when output is piped or redirected, unless explicitly forced.
- **Support `NO_COLOR`**
  - Honour the `NO_COLOR` environment variable for users who need plain output.
- **Minimize decorative noise**
  - Use borders and box-drawing characters sparingly.
  - Avoid large ASCII art banners and heavy animation, especially in `--help` or default output.
- **Provide “simple” output modes**
  - Consider flags like `--simple`, `--no-color`, or `--no-ascii` that:
    - Remove spinners, fancy borders, and complex layouts.
    - Use linear, screen-reader-friendly text or machine-readable formats (e.g. JSON).

For documentation & help:

- Keep `--help` output **boring, fast, and predictable**.
- Prefer standard man-page style structure where possible.
- Ensure examples are clear and copy-pastable.

---

## 7. Scriptability & Non-Interactive Mode

Whenever working on TUI/CLI behaviour:

- Ensure there is a **non-interactive path** for automation:
  - Flags / subcommands that perform the same operations without a TUI.
  - Stable, machine-parsable output mode where it makes sense.
- Use:
  - **Meaningful exit codes** (0 success, non-zero for distinct failure classes where appropriate).
  - **Consistent output formats** so integrations do not break easily.

When refactoring:

- Preserve CLI argument compatibility unless explicitly changing a public API.
- If changing behaviour, call out:
  - What breaks.
  - Migration path / compatibility flags if available.

---

## 8. Design Workflow When Using This Skill

When the user asks for help with TUI / CLI design:

1. **Clarify scope**
   - Is this a new tool, a refactor, or UX polish for an existing one?
   - Is the primary interface CLI, TUI, or both?
2. **Choose interaction model**
   - Decide which parts remain pure CLI and which, if any, need TUI.
3. **Define AppState and screens**
   - List screens / modes and key state fields.
4. **Plan keybindings and behaviour**
   - Start from the conventional bindings and add only what is needed.
5. **Check accessibility and automation**
   - Plan flags / modes for simple output, no colour, and non-interactive use.
6. **Propose architecture**
   - Present a layered plan (Core, UI, IO/event loop) that is easy to test and evolve.

---

## 9. Review Checklist

Before finalizing TUI/CLI design recommendations, verify:

- [ ] The interface **does few things, but does them well**.
- [ ] Behaviour is **deterministic** for given inputs and state.
- [ ] All operations are reachable via **keyboard** and keybindings follow common conventions or are clearly documented.
- [ ] **Core / UI / IO** responsibilities are cleanly separated.
- [ ] `AppState` is explicit and sufficient to render all screens.
- [ ] There is a **non-interactive** path for automation when appropriate.
- [ ] Output and help text respect **accessibility** concerns (colour, decoration, speed).
- [ ] Startup and exit paths are **fast** and do not surprise users.


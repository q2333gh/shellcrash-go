# How to Build Beautiful Terminal User Interfaces in Python（本地存档）

> 原文链接：<https://blog.tng.sh/2025/10/how-to-build-beautiful-terminal-user.html>  
> 说明：下文为原文内容的本地存档，便于离线阅读与搜索。

How to Build Beautiful Terminal User Interfaces in Python

# How to Build Beautiful Terminal User Interfaces in Python

## How to Build Beautiful Terminal User Interfaces in Python

To build a beautiful and maintainable Terminal User Interface (TUI) in Python, combine the `rich` library for vibrant presentation and the `questionary` library for interactive prompts. The key is to create a centralized theme class for styling (colors, icons, layout) and a base UI class with reusable components, ensuring a consistent and professional look across your entire application.

This guide breaks down the architecture used by the `tng-python` CLI to create its powerful interactive interface.

## What are `rich` and `questionary`?

The foundation of a modern Python TUI rests on two key libraries that handle presentation and interaction separately.

- **`rich`**: A library for writing rich text and beautiful formatting to the terminal. It's used for rendering panels, tables, styled text, progress bars, and more. It handles the *output*.

- **`questionary`**: A library for building interactive command-line prompts. It's used for asking questions, creating menus, and getting user input. It handles the *input*.

By combining them, you get a full-featured, app-like experience directly in the terminal.

## How to Structure a Theming System for TUIs

The most critical architectural decision for a scalable TUI is to **centralize all styling**. In the `tng-python` project, this is handled by a single `TngTheme` class in `theme.py`. This class acts as a single source of truth for all visual elements.

A well-structured theme class should contain nested classes for different aspects of styling:

1. **Colors**: Define all color names and styles (e.g., `PRIMARY`, `SUCCESS`, `TEXT_MUTED`).

2. **Icons**: Keep all Unicode icons/emojis in one place (e.g., `BACK`, `SUCCESS`, `FILE`).

3. **Layout**: Specify dimensions like padding, widths, and box styles.

4. **TextStyles**: Define semantic text styles (e.g., `TITLE`, `HEADER`, `INSTRUCTION`).

```python

class SomeClass:

"""Centralized theme configuration forPython UI"""

# ==================== COLORS ====================

class Colors:

PRIMARY = "cyan"

SUCCESS = "bold green"

WARNING = "bold yellow"

TEXT_MUTED = "dim white"

BORDER_DEFAULT = "cyan"

# ==================== ICONS ====================

class Icons:

BACK = "←"

SUCCESS = "✅"

FILE = "📄"

LIGHTBULB = "💡"

# ==================== LAYOUT ====================

class Layout:

PANEL_PADDING = (1, 2)

PANEL_BOX_STYLE = DOUBLE # from rich.box

# ==================== TEXT STYLES ====================

class TextStyles:

TITLE = "bold cyan"

HEADER = "bold white"

INSTRUCTION = "dim"

```

**Our unique insight:** The most effective TUIs separate presentation logic (the 'what') from styling (the 'how'). A centralized theme class is the architectural pattern that enables this separation, making complex UIs maintainable and easy to re-brand.

## How to Create Reusable UI Components

To avoid repeating code, create a `BaseUI` class that all your UI "screens" can inherit from. This base class, seen in `base_ui.py`, should:

1. Initialize the `Console` from `rich` and your `TngTheme`.

2. Provide helper methods for creating common UI elements like styled panels.

The `create_centered_panel` method is a perfect example. It takes content and a title, applies consistent styling from the theme, and returns a `Panel` object ready to be displayed.

```python

from rich.console import Console

from rich.panel import Panel

from rich.align import Align

from .theme import TngTheme

class BaseUI:

def __init__(self):

self.console = Console()

self.theme = TngTheme()

def create_centered_panel(self, content, title, border_style=None):

"""Create a centered panel with consistent styling"""

if border_style is None:

border_style = self.theme.Colors.BORDER_DEFAULT

panel = Panel(

Align.center(content),

title=title,

border_style=border_style,

padding=self.theme.Layout.PANEL_PADDING,

box=self.theme.Layout.PANEL_BOX_STYLE

)

return panel

```

By using this helper, every panel in the application looks the same, reinforcing a professional user experience.

## How to Build Interactive Prompts and Menus

`questionary` makes it easy to build interactive menus. The key is to integrate your theme with `questionary`'s styling system. You can create a method in your theme class that returns a `questionary.Style` object.

```python

# In theme.py

import questionary

class Theme:

# ... (other classes) ...

@classmethod

def get_questionary_style(cls):

"""Get questionary style configuration"""

return questionary.Style([

('question', cls.TextStyles.QUESTION), # "bold cyan"

('answer', cls.TextStyles.ANSWER), # "bold green"

('pointer', cls.Colors.POINTER), # "bold yellow"

('highlighted', cls.Colors.HIGHLIGHTED), # "bold green"

('selected', cls.Colors.SELECTED), # "bold green"

])

```

Then, in your UI screens, you pass this style to your prompts.

```python

# In any UI screen class

import questionary

from .base_ui import BaseUI

class MyScreen(BaseUI):

def show_menu(self):

action = questionary.select(

"Select an option:",

choices=["Generate Tests", "View Stats", "Exit"],

style=self.theme.get_questionary_style() # Use the centralized theme

).ask()

return action

```

This ensures even interactive elements match your application's brand.

## How to Display Rich Content like Tables

The `rich` library excels at displaying structured data. The `Table` class lets you create beautiful, formatted tables with headers, titles, and custom styles drawn directly from your theme.

```python

# A simplified example from help_ui.py

from rich.table import Table

class HelpUI(BaseUI):

def show(self):

table = Table(

title="Available Commands",

show_header=True,

header_style=self.theme.TextStyles.HEADER,

box=self.theme.Layout.PANEL_BOX_STYLE

)

table.add_column("Command", style=self.theme.Colors.PRIMARY)

table.add_column("Description", style=self.theme.Colors.TEXT_PRIMARY)

table.add_row("tng", "Start interactive test generation mode")

table.add_row("tng-init", "Generate TNG configuration file")

self.console.print(table)

```

## FAQ for Building Python TUIs

### What are the best libraries for Python TUIs in 2025?

For CLI applications, the combination of **`rich`** (for display) and **`questionary`** (for prompts) is a powerful, modern, and easy-to-use stack. For full-screen, app-like experiences with more complex layouts and widgets, **`textual`** (from the creator of `rich`) is the leading choice.

### Is `rich` better than the built-in `curses` library?

Yes, for most use cases. `curses` is a low-level library that requires manual handling of screen state, positioning, and colors, which is complex and error-prone. `rich` provides a high-level, declarative API that handles all the hard parts for you, making it significantly faster to develop and maintain beautiful TUIs.

### How do you handle terminal width and responsiveness?

The `rich.console.Console` object can automatically detect the terminal width. You can build responsive layouts by creating `Panel` and `Table` objects whose dimensions are based on the console width, as demonstrated in the `TngTheme`'s `center_text` and `calculate_box_width` helper methods.

### Can you use emojis and icons in the terminal?

Yes. Modern terminals fully support Unicode, including emojis. Storing them in a central `Icons` class within your theme (like in `theme.py`) makes them easy to manage and use consistently. They add significant visual appeal and clarity to a TUI.

### What's the difference between `rich` and `textual`?

`rich` is a library for rendering rich content in the terminal. You print something, and it's done. `textual` is a full application framework for building TUIs. It runs an event loop, manages state, and has a widget-based system similar to a GUI framework. Use `rich` for CLI *output*; use `textual` for CLI *apps*. `tng-python` uses `rich` and `questionary` for its interactive prompts, which is a common pattern.

### How do you manage colors and styles consistently?

The best practice is to define all colors and semantic styles (like "title" or "error message") in a single, centralized `theme.py` file. In your UI code, always reference these theme variables (e.g., `self.theme.Colors.SUCCESS`) instead of hardcoding color strings like `"green"`. This allows you to change your entire application's look from one file.

From our repo: [https://tng.sh](https://tng.sh)


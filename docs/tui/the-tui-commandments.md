# The TUI Commandments（本地存档）

> 原文链接：<https://bczsalba.com/post/the-tui-commandments>  
> 说明：下文为原文内容的本地存档，便于离线阅读与搜索。

The terminal deserves its own UI paradigm. | BIC various large scale open source projects. Currently working at Drop & Render." /

[BIC: Information Compendium](https://bczsalba.com/)

Dark mode [Administration](https://bczsalba.com/admin)

● [Posts](https://bczsalba.com/)/ [Terminal](https://bczsalba.com/?q=terminal%2F)/

It is finally time for the terminal to receive its own paradigm. Not a spruced up version of its predecessor, not a worse version of its successor.

Something truly unique.

bczsalba.com/post/the-tui-commandments

Published

2025-08-20 12:27:25

Modified

2025-10-17 15:32:57

OUTLINE:

The lost ideal

The TUI Commandments

1. The program must do few things, but do them exceptionally.

2. The program must provide a deterministic interface.

3. The program must treat the singular keypress as the basic unit of interaction.

4. The program's ergonomics must be user configurable.

5. The program must make use of command line arguments where possible.

6. The program must launch and terminate immediately, with easy state resumption.

7. The program must avoid reimplementing terminal behaviour.

8. The program must use the TUI with purpose

The future

# The terminal deserves its own UI paradigm.

Terminals have a reputation for being both slow and difficult to program, on top of compatibility issues that put Internet Explorer to shame. This has thankfully been changing for the last decade: most terminals now adhere to the reasonable subset of the VT100 & xterm specs, alongside some "nice to have" features like bitmapped graphics and ostentatious performance (at least when comparing to older implementations). TUI libraries and frameworks have gotten easier to use, and thus the "TUI revolution" began.

Most of these tools focus on bringing a web-like experience to the terminal. This is impressive, for sure, but it is also kind of like trying to watch videos on a "color" E Ink screen. It ignores the natural advantages of the platform, only to create an at best equal (though usually worse) experience than that of the medium it is trying to mimic.

With all the historical bottlenecks now solved it is finally time for the terminal to receive its own paradigm. Not a spruced up version of its predecessor, not a worse version of its successor.

Something truly unique.

## The lost ideal

Terminal User Interfaces (TUIs, for short) were most popular in the years where Command Line Interfaces (CLIs) were no longer adequate, but Graphical User Interfaces (GUIs) weren't feasible yet. As a result most TUIs occupy the space in between, not being nearly as nice to use as a native GUI, but also lacking a lot of the aspects that made CLIs so efficient.

Working on TUI libraries for over half a decade now has given me a ton of perspective and insight into how they currently function, both in the positive and negative aspects. I strongly believe the medium never got the attention it really deserved, and that there is a world in which TUIs occupy a space nothing else can. The following are a set of guidelines to steer us into this new world.

## The TUI Commandments

### 1. The program must do few things, but do them exceptionally.

The tiny footprint of TUIs - and the lack of hardware requirements - means unique pieces of software can be written to handle specific tasks. Specialized tools that do one thing well are always to be preferred over jacks of all trade that sacrifice utility and usability for more features.

### 2. The program must provide a deterministic interface.

The same sets of initial conditions and subsequent input should always result in the same layout appearing. Interfaces should be treated as two parts; the elements that make up the "Inputs", and the "Content" they control. Inputs shouldn't change positions unless necessary, and in-application scrolling should be avoided in favor of native scrollback, if at all possible. In the case where an interface is loaded dynamically, the Inputs' position and label should remain consistent between updates, and only the Content should appear to change.

### 3. The program must treat the singular keypress as the basic unit of interaction.

Programs in the terminal have the unique assumption that all users have access to a physical* keyboard at all times. A user must be able to access the entire program with only the keyboard. Every action must be considered keyboard-first, and any other forms of access (e.g. mouse input) should explicitly be secondary. Direct inputs (keybinds) always trump relative ones (arrows between menu options), and the latter's necessity should be minimized by design.

*: This assumption is partially broken by terminal emulators running on mobile devices. These usually provide the full on-screen keyboard unless dismissed, but modifier keys and specific placement often varies.

### 4. The program's ergonomics must be user configurable.

A great TUI should come with a remappable set of ergonomic defaults. It should be expected that some users will not have access to the intended keys due to keyboard layout differences or terminal-level remaps. Every key binding that could be user configured should be user configured, and this configuration should be done in a shareable, human readable & editable format. Visual style can also be configurable, but a cohesive default experience should always be preferred over the "great if you configure it well" approach.

### 5. The program must make use of command line arguments where possible.

The language of the terminal is spoken through CLI arguments. These should be used as possible to pre-select menus, set one-time configuration values or provide data through pipes. While full automation through static commands is often not feasible, programs with such capabilities are to be looked at as inspiration.

### 6. The program must launch and terminate immediately, with easy state resumption.

TUIs are usually not meant to be long-running processes. Users are expected to come in, complete their tasks and leave immediately afterwards. Startup speeds should be minimized, while "loading skeletons" are to be avoided at all costs. The key metric is time-to-task-completion, which describes the average amount of time it takes to go from the shell command prompt, complete one of the "Intended Tasks", and return back out of the program.

### 7. The program must avoid reimplementing terminal behaviour.

A program that users are expected to copy text out of should opt out of handling mouse drag inputs, so the native drag-to-select behaviour can take over - a program that primarily displays a single, long document should use the native scrollback instead of implementing its own virtualized scrolling, so it still scrolls correctly and performantly in multiplexers like TMUX - and so on. Always strive to work with the platform, rather than against it.

### 8. The program must use the TUI with purpose

TUIs are written with the same tooling CLIs are. Efficiency is king - if something can be expressed using a single-line CLI expression it will always be faster, thus a better choice, to interact with than a TUI wrapping the same functionality. Avoid TUIs for single-step processes and non-interactive getters, but feel free to use them when the context becomes more complicated. TUIs should ideally start out as CLIs, only adding new interactive interfaces as necessitated by complexity.

## The future

Going forward, all of my actively maintained TUI tools will follow this standard. I am already implementing points 2, 3 and 4 into [Celadon](https://bczsalba.com/post/celadon) with a system I call QuickSelect™. Every program written using Celadon will support QuickSelect™, and thus be in-line with almost half of all commandments. At this moment I don't plan on bringing it to the legacy [PyTermGUI](https://bczsalba.com/post/pytermgui).

The terminal is a great platform, but it's long been held down by the question of "Why is this a TUI?". I hope these standards (or a variation of them; I don't particularly care as long as the ideas get through) can finally make the TUI make sense.

### Comments

---

[Leave a comment ↗](mailto:mail@bczsalba.com?subject=RE%3A%20the-tui-commandments&body=Please%20enter%20your%20comment%20below%20the%20divider%20%28%27---%27%29.%0AAn%20empty%20message%20aborts%20the%20comment%2C%20but%20a%20name%20change%20will%20still%20apply.%20You%20may%20use%20markdown.%0A%0AYour%20current%20email%20account%27s%20first%20name%20will%20be%20used%20to%20display%20your%20comment.%20To%20change%20this%2C%20please%20replace%20%27default%27%20with%20your%20chosen%20name.%0A%0ANAME%3A%20default%0A---) [(Open email template)](https://bczsalba.com/email-template/comment/the-tui-commandments)

### Related

---

● [Posts](https://bczsalba.com/)/ [Terminal](https://bczsalba.com/?q=terminal%2F)/

Python TUI framework with mouse support, modular widget system, customizable and a rapid terminal markup language.

bczsalba.com/post/pytermgui

Published

2025-03-27 12:25:43

Modified

2025-10-07 22:07:21

| [PyTermGUI: TUIs that don't suck ↗](https://bczsalba.com/post/pytermgui) |
| --- |

● [Posts](https://bczsalba.com/)/ [Terminal](https://bczsalba.com/?q=terminal%2F)/

Sipedon is an aquarium simulator that runs in your terminal. It is currently in its early stages, but eventually it should become an aquarium management game.

bczsalba.com/post/sipedon

Published

2025-03-27 12:10:48

Modified

2025-05-26 22:09:18

| [Sipedon: An aquarium for your terminal ↗](https://bczsalba.com/post/sipedon) |
| --- |


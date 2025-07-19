

# <div align="center">💻</div>
<div align="center"><img src="https://github.com/non-erx/spv/blob/main/pics/spvlogo.png?raw=true" alt="spv logo" width="7800"></div>
<div  align="center"><sup>Screen Process Viewer (spv)</sup></div>

<div  align="center">A modern, elegant TUI for managing your GNU Screen sessions.</div>

<div  align="center"><i>Made with love by @non-erx!❤️</i></div>

<br>

`spv` is a terminal-based tool written in Go that provides a beautiful and responsive two-pane interface to monitor, manage, and interact with all your `screen` sessions. It's designed to be fast, intuitive, and highly customizable.

### 🚀 Installation

You can install `spv` in one of three ways:

**1. Using `go install` (Recommended)**

If you have Go installed, this is the easiest method:
```bash
go install github.com/non-erx/spv@latest
```

**2. From Source**

Clone the repository and build the binary yourself:
```bash
git clone https://github.com/non-erx/spv.git
cd spv
go build -ldflags="-s -w" -o spv
```

### 💡 Usage

Simply run the binary from your terminal:
```bash
./spv
```

#### Keybindings

All keybindings are conveniently displayed in the footer of the application:

| Key | Action |
| :--- | :--- |
| **↑↓** | Navigate through the session list |
| **Enter** | Attach to the selected session |
| **a** | Add a new session |
| **k** | Kill the selected session |
| **r** | Refresh the session list and stats |
| **?** | Show the about screen |
| **q** | Quit the application |

#### 🎨 Theming

`spv` comes with a few built-in themes. To set a theme and save it as your default, run:
```bash
./spv theme <theme_name>
```
**Available Themes:** `slate` (default), `pink`, `forest`.

### 🌠 Screenshots
<div align="center"><img src="https://github.com/non-erx/spv/blob/main/pics/tui_slate.png?raw=true" alt="spv slate" width="2500"></div>
<div align="center"><img src="https://github.com/non-erx/spv/blob/main/pics/tui_pink.png?raw=true" alt="spv pink" width="2500"></div>
<div align="center"><img src="https://github.com/non-erx/spv/blob/main/pics/tui_forest.png?raw=true" alt="spv forest" width="2500"></div>
### ✨ Features

-   `🖥️` **Elegant TUI:** A beautiful and responsive two-pane interface for at-a-glance information, built with Bubble Tea.
-   `🔄` **Live Data:** Auto-refreshes every second with real-time CPU, RAM, and session status updates.
-   `🚀` **Dynamic Header:** Displays the latest commit message from this GitHub repository, keeping you in the loop.
-   `🎨` **Customizable Themes:** Choose from multiple built-in themes (`slate`, `pink`, `forest`) and save your preference.
-   `💾` **Persistent Sessions:** Remembers session commands and descriptions across restarts via a simple JSON configuration file.
-   `⚡` **Autostart Configuration:** Easily flag sessions to be started on system reboot (requires a user-side script to read `~/.config/spv/autostart.json`).
-   `📜` **Detailed View:** See a session's ID, status (Attached/Detached), autostart configuration, the command it's running, and a custom description.
-   `⌨️` **Intuitive Workflow:** A multi-step wizard guides you through creating new sessions (Name → Command → Description → Autostart).
<div  align="center">
-
<div  align="center">
<sub >Check out my silly web-page -> <a href="https://non-erx.dev">non-erx.dev </a></sub>
</div>

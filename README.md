# wt - Git Worktree + Tmux Manager

A CLI tool that provides seamless integration between git worktrees and tmux sessions.

## Features

- **Create worktrees** with automatic tmux session setup
- **Switch between worktrees** (and their tmux sessions)
- **Delete worktrees** with session cleanup
- **Toggle panes** - hide/show panes like Claude with keybindings
- **Static panes** - permanent panes for testing, servers, etc.
- **Pane hooks** - run commands when panes are opened/closed
- **Share files** across worktrees via symlinks or copies
- **Post-create hooks** to run commands after creating worktrees
- **Shell completions** for bash, zsh, fish

## Installation

```bash
# Clone and build
git clone https://github.com/yunus/wt.git
cd wt
make install

# Or build only
make build
./build/wt --help
```

## Quick Start

```bash
# Initialize global config
wt init

# Navigate to a git bare repo
cd ~/repos/my-project

# Create a new worktree
wt add feature-x

# List all worktrees
wt list

# Switch to a worktree
wt switch feature-x

# Toggle Claude pane (show/hide)
wt toggle claude

# Delete a worktree
wt delete feature-x
```

## Configuration

### Global Config: `~/.config/wt/config.yaml`

```yaml
window_prefix: "wt-"

# Pane layout for worktree sessions
panes:
  # Main editor pane
  - name: editor
    command: nvim .
    focus: true

  # Shell pane (split horizontally)
  - name: shell
    split: horizontal

  # Toggleable Claude pane (can be hidden/shown)
  - name: claude
    command: claude
    toggle: true
    position: left
    size: 20
    hooks:
      on_open: "echo 'Claude opened'"
      on_close: "echo 'Claude closed'"

# Commands to run after creating worktrees
post_create:
  - npm install

# Share files across worktrees
files:
  symlink:
    - node_modules
    - .env

agent: claude
```

### Project Config: `.wt.yaml`

Create a project-specific config by running `wt init` in your git repo:

```yaml
# Use <global> to include global config values
post_create:
  - '<global>'
  - npm run db:migrate

files:
  symlink:
    - '<global>'
    - .next/cache
  copy:
    - .env.local

panes:
  - name: editor
    command: nvim .
    focus: true
  - name: server
    command: npm run dev
    split: horizontal
  - name: test
    command: npm test --watch
    split: vertical
  - name: claude
    command: <agent>
    toggle: true
    position: left
    size: 20
```

## Pane Types

### Static Panes

Static panes are always visible and part of the main window. Configure them without `toggle: true`:

```yaml
panes:
  - name: editor
    command: nvim .
    focus: true
  - name: server
    command: npm run dev
    split: horizontal
```

### Toggleable Panes

Toggleable panes can be hidden/shown with the `wt toggle` command:

```yaml
panes:
  - name: claude
    command: claude
    toggle: true       # Makes this pane toggleable
    position: left     # "left" or "right"
    size: 20           # Percentage width
    hooks:
      on_open: "echo 'Pane opened'"
      on_close: "echo 'Pane closed'"
```

Toggle with:
```bash
wt toggle claude
```

Or bind to a tmux key:
```bash
# In ~/.tmux.conf
bind C run-shell "wt toggle claude"
```

## Shell Completions

```bash
# Bash (~/.bashrc)
eval "$(wt completions bash)"

# Zsh (~/.zshrc)
eval "$(wt completions zsh)"

# Fish (~/.config/fish/config.fish)
wt completions fish | source
```

## Tmux Integration

Add these keybindings to your `~/.tmux.conf`:

```bash
# ===== WORKTREE MANAGEMENT (wt CLI) =====
bind g run-shell "wt toggle >/dev/null 2>&1"
bind T display-popup -E -w 60% -h 50% "wt switch"
bind B command-prompt -p "new branch:" "run-shell 'wt add %% --no-switch >/dev/null 2>&1'"
```

Then reload: `tmux source-file ~/.tmux.conf`

### Key Bindings

| Binding | Action |
|---------|--------|
| `prefix + g` | Toggle Claude pane (20% on right) |
| `prefix + T` | Switch worktree (interactive popup) |
| `prefix + B` | Create new worktree (prompts for branch) |

### Picker Navigation

The interactive picker (`prefix + T`) supports:
- `↑/↓` or `ctrl+n/p` - Navigate (wraps around)
- Type to fuzzy filter
- `Enter` - Select
- `Esc` - Cancel

## Commands

| Command | Description |
|---------|-------------|
| `wt add <branch>` | Create a new worktree and tmux session |
| `wt list` | List all worktrees |
| `wt switch <name>` | Switch to a worktree |
| `wt delete <name>` | Delete a worktree and its session |
| `wt toggle <pane>` | Toggle a pane's visibility |
| `wt toggle --list` | List all toggleable panes |
| `wt init` | Initialize configuration |
| `wt completions <shell>` | Generate shell completions |

## Configuration Reference

### Pane Options

| Option | Type | Description |
|--------|------|-------------|
| `name` | string | Identifier for the pane |
| `command` | string | Command to run (supports `<agent>` placeholder) |
| `split` | string | Split direction: "horizontal" or "vertical" |
| `position` | string | Position for toggle panes: "left" or "right" |
| `size` | int | Size as percentage |
| `focus` | bool | Focus this pane on startup |
| `toggle` | bool | Can this pane be toggled? |
| `hooks.on_open` | string | Command to run when pane is shown |
| `hooks.on_close` | string | Command to run when pane is hidden |

### Files Options

```yaml
files:
  symlink:              # Create symlinks (saves disk space)
    - node_modules
    - .env
  copy:                 # Copy files (independent copies)
    - .env.local
```

### The `<global>` Marker

In project configs, use `<global>` to include values from the global config:

```yaml
post_create:
  - '<global>'          # Include all global post_create commands
  - npm run lint        # Add project-specific command
```

### The `<agent>` Placeholder

Use `<agent>` in pane commands to use the configured AI agent:

```yaml
agent: claude           # Set the agent command

panes:
  - name: ai
    command: <agent>    # Will expand to "claude"
    toggle: true
```

## Requirements

- Go 1.21+
- tmux
- Git with bare repo + worktree setup

## License

MIT

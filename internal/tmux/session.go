package tmux

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/yunus/wt/internal/config"
)

var (
	ErrNotInTmux       = errors.New("not running inside tmux")
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExists   = errors.New("session already exists")
)

// IsInsideTmux returns true if we're running inside a tmux session
func IsInsideTmux() bool {
	return os.Getenv("TMUX") != ""
}

// GetCurrentSession returns the current tmux session name
func GetCurrentSession() (string, error) {
	if !IsInsideTmux() {
		return "", ErrNotInTmux
	}
	cmd := exec.Command("tmux", "display-message", "-p", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// GetCurrentWindow returns the current window index
func GetCurrentWindow() (string, error) {
	if !IsInsideTmux() {
		return "", ErrNotInTmux
	}
	cmd := exec.Command("tmux", "display-message", "-p", "#{window_index}")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// GetCurrentWindowName returns the current window name
func GetCurrentWindowName() (string, error) {
	if !IsInsideTmux() {
		return "", ErrNotInTmux
	}
	cmd := exec.Command("tmux", "display-message", "-p", "#{window_name}")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// SessionExists checks if a tmux session exists
func SessionExists(name string) bool {
	cmd := exec.Command("tmux", "has-session", "-t", name)
	return cmd.Run() == nil
}

// CreateSession creates a new tmux session with windows from config
func CreateSession(name, path string, cfg *config.Config) error {
	if SessionExists(name) {
		return ErrSessionExists
	}

	if len(cfg.Windows) == 0 {
		// Fallback: create a simple session with one window
		args := []string{"new-session", "-d", "-s", name, "-c", path}
		cmd := exec.Command("tmux", args...)
		return cmd.Run()
	}

	// Create session with first window
	firstWindow := cfg.Windows[0]
	windowName := cfg.WindowPrefix + firstWindow.Name
	args := []string{"new-session", "-d", "-s", name, "-n", windowName, "-c", path}
	cmd := exec.Command("tmux", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Setup panes in first window
	setupWindowPanes(name, windowName, path, firstWindow.Panes, cfg)

	// Create additional windows
	for i, window := range cfg.Windows[1:] {
		windowName := cfg.WindowPrefix + window.Name
		if err := createWindowWithPanes(name, i+1, windowName, path, window.Panes, cfg); err != nil {
			fmt.Printf("Warning: failed to create window %s: %v\n", window.Name, err)
		}
	}

	// Create toggle pane window (hidden, will be joined when toggled)
	if cfg.HasTogglePane() {
		createTogglePaneWindow(name, path, cfg)
	}

	// Select the focused window
	for i, window := range cfg.Windows {
		if window.Focus {
			windowName := cfg.WindowPrefix + window.Name
			SelectWindow(name, strconv.Itoa(i))
			// Also select first pane in that window
			SelectPaneInWindow(name, windowName, 0)
			break
		}
	}

	return nil
}

// createWindowWithPanes creates a window with its panes
func createWindowWithPanes(session string, index int, windowName, path string, panes []config.Pane, cfg *config.Config) error {
	// Create the window
	target := session + ":" + strconv.Itoa(index)
	cmd := exec.Command("tmux", "new-window", "-t", target, "-n", windowName, "-c", path)
	if err := cmd.Run(); err != nil {
		return err
	}

	// Setup panes
	setupWindowPanes(session, windowName, path, panes, cfg)
	return nil
}

// setupWindowPanes creates panes within a window
func setupWindowPanes(session, windowName, path string, panes []config.Pane, cfg *config.Config) {
	if len(panes) == 0 {
		return
	}

	// First pane already exists, run its command
	if panes[0].Command != "" {
		cmd := cfg.ExpandAgentPlaceholder(panes[0].Command)
		SendKeys(session, windowName, cmd)
	}

	// Create additional panes
	target := session + ":" + windowName
	for i, pane := range panes[1:] {
		paneIdx := i + 1

		// Determine split direction
		splitFlag := "-h" // horizontal (side by side)
		if pane.Split == "vertical" {
			splitFlag = "-v" // vertical (stacked)
		}

		// Split pane
		splitArgs := []string{"split-window", splitFlag, "-t", target, "-c", path}
		if pane.Size > 0 {
			splitArgs = append(splitArgs, "-l", strconv.Itoa(pane.Size)+"%")
		}
		splitCmd := exec.Command("tmux", splitArgs...)
		if err := splitCmd.Run(); err != nil {
			fmt.Printf("Warning: failed to split pane: %v\n", err)
			continue
		}

		// Run command in new pane
		if pane.Command != "" {
			cmd := cfg.ExpandAgentPlaceholder(pane.Command)
			SendKeysToPane(session, windowName, paneIdx, cmd)
		}
	}

	// Select first pane
	SelectPaneInWindow(session, windowName, 0)
}

// createTogglePaneWindow creates the hidden window for the toggle pane
func createTogglePaneWindow(session, path string, cfg *config.Config) {
	if cfg.TogglePane == nil {
		return
	}

	hiddenWindowName := "_" + cfg.TogglePane.Name

	// Get window count to determine index
	windows, _ := ListWindows(session)
	newIdx := len(windows)

	target := session + ":" + strconv.Itoa(newIdx)
	cmd := exec.Command("tmux", "new-window", "-d", "-t", target, "-n", hiddenWindowName, "-c", path)
	if err := cmd.Run(); err != nil {
		fmt.Printf("Warning: failed to create toggle pane window: %v\n", err)
		return
	}

	// Run command in toggle pane
	if cfg.TogglePane.Command != "" {
		command := cfg.ExpandAgentPlaceholder(cfg.TogglePane.Command)
		SendKeys(session, hiddenWindowName, command)
	}
}

// SwitchSession switches to a session (attach or switch-client based on context)
func SwitchSession(name string) error {
	if !SessionExists(name) {
		return ErrSessionNotFound
	}

	if IsInsideTmux() {
		// Inside tmux: use switch-client
		cmd := exec.Command("tmux", "switch-client", "-t", name)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("switch-client failed: %w (output: %s)", err, string(output))
		}
		return nil
	}

	// Outside tmux: attach to session
	cmd := exec.Command("tmux", "attach-session", "-t", name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// KillSession terminates a tmux session
func KillSession(name string) error {
	if !SessionExists(name) {
		return nil // Already doesn't exist
	}
	cmd := exec.Command("tmux", "kill-session", "-t", name)
	return cmd.Run()
}

// ListSessions returns all tmux sessions
func ListSessions() ([]string, error) {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var sessions []string
	for _, line := range lines {
		if line != "" {
			sessions = append(sessions, line)
		}
	}
	return sessions, nil
}

// SendKeysToPane sends keys to a specific pane
func SendKeysToPane(session, window string, paneIdx int, keys string) error {
	target := fmt.Sprintf("%s:%s.%d", session, window, paneIdx)
	cmd := exec.Command("tmux", "send-keys", "-t", target, keys, "Enter")
	return cmd.Run()
}

// SelectPaneInWindow selects a pane in a specific window
func SelectPaneInWindow(session, window string, paneIdx int) error {
	target := fmt.Sprintf("%s:%s.%d", session, window, paneIdx)
	cmd := exec.Command("tmux", "select-pane", "-t", target)
	return cmd.Run()
}

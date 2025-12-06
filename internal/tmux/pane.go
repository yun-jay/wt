package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/yunus/wt/internal/config"
)

// CountPanes returns the number of panes in the current window
func CountPanes() (int, error) {
	session, err := GetCurrentSession()
	if err != nil {
		return 0, err
	}
	window, err := GetCurrentWindow()
	if err != nil {
		return 0, err
	}

	return CountPanesInWindow(session, window)
}

// CountPanesInWindow returns the number of panes in a specific window
func CountPanesInWindow(session, window string) (int, error) {
	cmd := exec.Command("tmux", "list-panes", "-t", session+":"+window)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	count := 0
	for _, line := range lines {
		if line != "" {
			count++
		}
	}
	return count, nil
}

// BreakPane breaks a pane out to its own window
func BreakPane(session, sourceWindow string, paneIndex int, newWindowName string) error {
	source := session + ":" + sourceWindow + "." + strconv.Itoa(paneIndex)
	cmd := exec.Command("tmux", "break-pane", "-d", "-s", source, "-n", newWindowName)
	return cmd.Run()
}

// JoinPane joins a pane from another window to the current window
func JoinPane(session, sourceWindow, targetWindow string, position string, sizePercent int) error {
	source := session + ":" + sourceWindow + ".0"
	target := session + ":" + targetWindow

	// Determine split direction based on position
	splitFlag := "-h" // horizontal (side by side) - right
	if position == "left" {
		splitFlag = "-hb" // horizontal before (left)
	}

	args := []string{"join-pane", splitFlag, "-l", strconv.Itoa(sizePercent) + "%", "-s", source, "-t", target}
	cmd := exec.Command("tmux", args...)
	return cmd.Run()
}

// SelectPane selects a pane in the current window
func SelectPane(paneIndex int) error {
	cmd := exec.Command("tmux", "select-pane", "-t", strconv.Itoa(paneIndex))
	return cmd.Run()
}

// Toggle toggles the configured toggle pane (show/hide in current window)
func Toggle(cfg *config.Config) error {
	if !IsInsideTmux() {
		return ErrNotInTmux
	}

	if cfg.TogglePane == nil {
		return fmt.Errorf("no toggle pane configured")
	}

	session, err := GetCurrentSession()
	if err != nil {
		return err
	}
	currentWindow, err := GetCurrentWindow()
	if err != nil {
		return err
	}

	toggleCfg := cfg.TogglePane
	hiddenWindowName := "_" + toggleCfg.Name

	// Check if the toggle pane is currently visible in this window
	if isTogglePaneVisible(session, currentWindow, toggleCfg.Name) {
		// Pane is visible - hide it (break out to hidden window)
		return hideTogglePane(session, currentWindow, hiddenWindowName, toggleCfg)
	}

	// Pane is hidden - show it (join to current window)
	return showTogglePane(session, currentWindow, hiddenWindowName, toggleCfg, cfg)
}

// isTogglePaneVisible checks if the toggle pane is in the current window
func isTogglePaneVisible(session, window, paneName string) bool {
	hiddenWindowName := "_" + paneName
	// If hidden window exists, the pane is NOT in the current window
	idx, _ := FindWindowByName(session, hiddenWindowName)
	return idx < 0
}

// hideTogglePane hides the toggle pane by breaking it out to a hidden window
func hideTogglePane(session, window, hiddenWindowName string, toggleCfg *config.TogglePaneConfig) error {
	// Run on_close hook first
	if toggleCfg.Hooks.OnClose != "" {
		runHook(toggleCfg.Hooks.OnClose)
	}

	// Find the pane index to break out based on position
	paneIdx := 0
	if toggleCfg.Position == "right" {
		// Get pane count and break the last one
		paneCount, _ := CountPanesInWindow(session, window)
		paneIdx = paneCount - 1
	}

	if err := BreakPane(session, window, paneIdx, hiddenWindowName); err != nil {
		return fmt.Errorf("failed to hide pane: %w", err)
	}

	// Select the main pane
	SelectPane(0)

	return nil
}

// showTogglePane shows the toggle pane by joining it to the current window
func showTogglePane(session, currentWindow, hiddenWindowName string, toggleCfg *config.TogglePaneConfig, cfg *config.Config) error {
	// Check if hidden window exists
	hiddenWindowIdx, err := FindWindowByName(session, hiddenWindowName)
	if err != nil || hiddenWindowIdx < 0 {
		// Need to create the toggle pane window first
		return createAndShowTogglePane(session, currentWindow, toggleCfg, cfg)
	}

	// Join the hidden pane to current window
	position := toggleCfg.Position
	if position == "" {
		position = "right"
	}
	size := toggleCfg.Size
	if size == 0 {
		size = 20
	}

	if err := JoinPane(session, strconv.Itoa(hiddenWindowIdx), currentWindow, position, size); err != nil {
		return fmt.Errorf("failed to show pane: %w", err)
	}

	// Run on_open hook
	if toggleCfg.Hooks.OnOpen != "" {
		runHook(toggleCfg.Hooks.OnOpen)
	}

	// Select the main pane (not the toggle pane)
	if position == "left" {
		SelectPane(1) // Main pane is now index 1
	} else {
		SelectPane(0) // Main pane stays at index 0
	}

	return nil
}

// createAndShowTogglePane creates a new toggle pane and shows it
func createAndShowTogglePane(session, currentWindow string, toggleCfg *config.TogglePaneConfig, cfg *config.Config) error {
	hiddenWindowName := "_" + toggleCfg.Name

	// Get window count to determine index
	windows, _ := ListWindows(session)
	newIdx := len(windows)

	// Create window
	target := session + ":" + strconv.Itoa(newIdx)
	cmd := exec.Command("tmux", "new-window", "-d", "-t", target, "-n", hiddenWindowName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create toggle pane window: %w", err)
	}

	// Run command in it
	if toggleCfg.Command != "" {
		command := cfg.ExpandAgentPlaceholder(toggleCfg.Command)
		SendKeys(session, hiddenWindowName, command)
		time.Sleep(500 * time.Millisecond)
	}

	// Now join it to current window
	hiddenWindowIdx, _ := FindWindowByName(session, hiddenWindowName)
	position := toggleCfg.Position
	if position == "" {
		position = "right"
	}
	size := toggleCfg.Size
	if size == 0 {
		size = 20
	}

	if err := JoinPane(session, strconv.Itoa(hiddenWindowIdx), currentWindow, position, size); err != nil {
		return fmt.Errorf("failed to show pane: %w", err)
	}

	// Run on_open hook
	if toggleCfg.Hooks.OnOpen != "" {
		runHook(toggleCfg.Hooks.OnOpen)
	}

	// Select the main pane
	if position == "left" {
		SelectPane(1)
	} else {
		SelectPane(0)
	}

	return nil
}

// runHook runs a hook command in the background
func runHook(command string) {
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

// IsTogglePaneVisible checks if the toggle pane is currently visible in any window
func IsTogglePaneVisible(cfg *config.Config) (bool, error) {
	if !IsInsideTmux() {
		return false, ErrNotInTmux
	}

	if cfg.TogglePane == nil {
		return false, nil
	}

	session, err := GetCurrentSession()
	if err != nil {
		return false, err
	}
	currentWindow, err := GetCurrentWindow()
	if err != nil {
		return false, err
	}

	return isTogglePaneVisible(session, currentWindow, cfg.TogglePane.Name), nil
}

package tmux

import (
	"os/exec"
	"strconv"
	"strings"
)

// Window represents a tmux window configuration
type Window struct {
	Name    string `yaml:"name"`
	Command string `yaml:"command,omitempty"`
}

// CreateWindow creates a new window in a session
func CreateWindow(session string, index int, name, path string) error {
	args := []string{"new-window", "-t", session + ":" + strconv.Itoa(index), "-n", name, "-c", path}
	cmd := exec.Command("tmux", args...)
	return cmd.Run()
}

// SelectWindow selects a window in a session
func SelectWindow(session, windowTarget string) error {
	cmd := exec.Command("tmux", "select-window", "-t", session+":"+windowTarget)
	return cmd.Run()
}

// SendKeys sends keys to a tmux target
func SendKeys(session, target, keys string) error {
	fullTarget := session + ":" + target
	cmd := exec.Command("tmux", "send-keys", "-t", fullTarget, keys, "Enter")
	return cmd.Run()
}

// FindWindowByName finds a window by name in a session and returns its index
func FindWindowByName(session, name string) (int, error) {
	cmd := exec.Command("tmux", "list-windows", "-t", session, "-F", "#{window_index}:#{window_name}")
	output, err := cmd.Output()
	if err != nil {
		return -1, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && parts[1] == name {
			idx, err := strconv.Atoi(parts[0])
			if err != nil {
				continue
			}
			return idx, nil
		}
	}

	return -1, nil
}

// RenameWindow renames a window
func RenameWindow(session, windowTarget, newName string) error {
	cmd := exec.Command("tmux", "rename-window", "-t", session+":"+windowTarget, newName)
	return cmd.Run()
}

// ListWindows returns all windows in a session
func ListWindows(session string) ([]string, error) {
	cmd := exec.Command("tmux", "list-windows", "-t", session, "-F", "#{window_name}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var windows []string
	for _, line := range lines {
		if line != "" {
			windows = append(windows, line)
		}
	}
	return windows, nil
}

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yunus/wt/internal/tmux"
)

var toggleCmd = &cobra.Command{
	Use:   "toggle",
	Short: "Toggle the Claude pane visibility",
	Long: `Toggle the configured toggle pane (e.g., Claude) in the current window.

When toggled ON: The pane appears on the configured side (left/right) of the current window.
When toggled OFF: The pane is hidden to its own window.

Configure the toggle pane in your config:

  toggle_pane:
    name: claude
    command: claude
    position: right
    size: 20
    hooks:
      on_open: "echo 'Claude opened'"
      on_close: "echo 'Claude closed'"`,
	RunE: runToggle,
}

func runToggle(cmd *cobra.Command, args []string) error {
	if !tmux.IsInsideTmux() {
		return fmt.Errorf("toggle can only be used inside tmux")
	}

	if !cfg.HasTogglePane() {
		return fmt.Errorf("no toggle pane configured. Add 'toggle_pane' to your config")
	}

	visible, _ := tmux.IsTogglePaneVisible(cfg)
	action := "Showing"
	if visible {
		action = "Hiding"
	}
	fmt.Printf("%s %s pane...\n", action, cfg.TogglePane.Name)

	return tmux.Toggle(cfg)
}

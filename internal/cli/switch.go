package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yunus/wt/internal/git"
	"github.com/yunus/wt/internal/tmux"
	"github.com/yunus/wt/internal/tui"
)

var switchCmd = &cobra.Command{
	Use:   "switch [worktree]",
	Short: "Switch to a worktree",
	Long: `Switch to a worktree by name, branch, or path.

If no argument is provided, an interactive picker will be shown.
If no tmux session exists for the worktree, one will be created automatically.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSwitch,
}

func runSwitch(cmd *cobra.Command, args []string) error {
	repo, err := git.FindRepo()
	if err != nil {
		return fmt.Errorf("failed to find git repository: %w", err)
	}

	var worktreeName string

	if len(args) == 0 {
		// Interactive mode - show picker
		worktrees, err := repo.ListWorktrees()
		if err != nil {
			return fmt.Errorf("failed to list worktrees: %w", err)
		}

		if len(worktrees) == 0 {
			return fmt.Errorf("no worktrees found")
		}

		// Build picker items
		items := make([]tui.Item, len(worktrees))
		for i, wt := range worktrees {
			desc := wt.Path
			if tmux.SessionExists(wt.SessionName()) {
				desc += " [tmux]"
			}
			items[i] = tui.Item{
				Name:        wt.Name(),
				Description: desc,
				Value:       wt,
			}
		}

		selected, err := tui.RunPicker("Switch to worktree:", items)
		if err != nil {
			return fmt.Errorf("picker failed: %w", err)
		}
		if selected == nil {
			return nil // User cancelled
		}

		worktreeName = selected.Name
	} else {
		worktreeName = args[0]
	}

	wt, err := repo.FindWorktree(worktreeName)
	if err != nil {
		return fmt.Errorf("worktree not found: %s", worktreeName)
	}

	sessionName := wt.SessionName()

	// Create session if it doesn't exist
	if !tmux.SessionExists(sessionName) {
		fmt.Printf("Creating tmux session '%s'...\n", sessionName)
		if err := tmux.CreateSession(sessionName, wt.Path, cfg); err != nil {
			return fmt.Errorf("failed to create tmux session: %w", err)
		}
	}

	// Switch to session
	if err := tmux.SwitchSession(sessionName); err != nil {
		return fmt.Errorf("failed to switch session: %w", err)
	}

	return nil
}

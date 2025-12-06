package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yunus/wt/internal/git"
	"github.com/yunus/wt/internal/tmux"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current worktree and session status",
	Long:  `Display information about the current worktree and tmux session.`,
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	repo, err := git.FindRepo()
	if err != nil {
		return fmt.Errorf("failed to find git repository: %w", err)
	}

	// Get current worktree
	currentWt, err := repo.GetCurrentWorktree()
	if err != nil {
		fmt.Println("Not in a worktree")
	} else {
		fmt.Printf("Current worktree: %s\n", currentWt.Name())
		fmt.Printf("  Branch: %s\n", currentWt.Branch)
		fmt.Printf("  Path: %s\n", currentWt.Path)

		sessionName := currentWt.SessionName()
		if tmux.SessionExists(sessionName) {
			fmt.Printf("  Tmux session: %s (active)\n", sessionName)
		} else {
			fmt.Printf("  Tmux session: %s (not running)\n", sessionName)
		}
	}

	// Show all worktrees summary
	worktrees, err := repo.ListWorktrees()
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	fmt.Printf("\nWorktrees: %d total\n", len(worktrees))

	// Count active tmux sessions
	activeSessions := 0
	for _, wt := range worktrees {
		if tmux.SessionExists(wt.SessionName()) {
			activeSessions++
		}
	}
	fmt.Printf("Active tmux sessions: %d\n", activeSessions)

	// Show toggle pane status
	if cfg.HasTogglePane() {
		visible, _ := tmux.IsTogglePaneVisible(cfg)
		fmt.Printf("\nToggle pane (%s): ", cfg.TogglePane.Name)
		if visible {
			fmt.Println("visible")
		} else {
			fmt.Println("hidden")
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yunus/wt/internal/git"
	"github.com/yunus/wt/internal/tmux"
	"github.com/yunus/wt/internal/tui"
)

var deleteForce bool

var deleteCmd = &cobra.Command{
	Use:     "delete [worktree]",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete a worktree",
	Long: `Delete a git worktree and its associated tmux session.

If no argument is provided, an interactive picker will be shown.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDelete,
}

func init() {
	deleteCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "skip confirmation and force delete")
}

func runDelete(cmd *cobra.Command, args []string) error {
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

		// Build picker items (exclude protected branches unless force)
		var items []tui.Item
		for _, wt := range worktrees {
			if !deleteForce && cfg.IsProtectedBranch(wt.Branch) {
				continue // Skip protected branches in picker
			}
			desc := wt.Path
			if tmux.SessionExists(repo.SessionName(&wt)) {
				desc += " [tmux]"
			}
			items = append(items, tui.Item{
				Name:        wt.Name(),
				Description: desc,
				Value:       wt,
			})
		}

		if len(items) == 0 {
			return fmt.Errorf("no worktrees available for deletion (all are protected)")
		}

		selected, err := tui.RunPicker("Delete worktree:", items)
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

	// Check if this is a protected branch
	if cfg.IsProtectedBranch(wt.Branch) && !deleteForce {
		return fmt.Errorf("cannot delete protected branch '%s'. Use --force to override", wt.Branch)
	}

	// Check if we're currently in this worktree
	currentWt, _ := repo.GetCurrentWorktree()
	isCurrentWorktree := currentWt != nil && currentWt.Path == wt.Path

	if isCurrentWorktree && !deleteForce {
		return fmt.Errorf("cannot delete current worktree. Switch to another worktree first or use --force")
	}

	// Confirm deletion
	if !deleteForce {
		fmt.Printf("Delete worktree '%s' at %s? [y/N]: ", wt.Name(), wt.Path)
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	sessionName := repo.SessionName(wt)

	// If we're in this worktree, switch to default first
	if isCurrentWorktree && tmux.IsInsideTmux() {
		defaultBranch, _ := repo.GetDefaultBranch()
		defaultWt, err := repo.FindWorktree(defaultBranch)
		if err != nil {
			return fmt.Errorf("cannot find default worktree to switch to")
		}

		fmt.Printf("Switching to '%s' before deleting...\n", defaultWt.Name())
		defaultSessionName := repo.SessionName(defaultWt)
		if !tmux.SessionExists(defaultSessionName) {
			if err := tmux.CreateSession(defaultSessionName, defaultWt.Path, cfg); err != nil {
				return fmt.Errorf("failed to create session for default branch: %w", err)
			}
		}
		if err := tmux.SwitchSession(defaultSessionName); err != nil {
			return fmt.Errorf("failed to switch to default branch: %w", err)
		}
	}

	// Kill tmux session
	if tmux.SessionExists(sessionName) {
		fmt.Printf("Killing tmux session '%s'...\n", sessionName)
		if err := tmux.KillSession(sessionName); err != nil {
			fmt.Printf("Warning: failed to kill session: %v\n", err)
		}
	}

	// Delete worktree
	fmt.Printf("Deleting worktree '%s'...\n", wt.Name())
	if err := repo.DeleteWorktree(wt.Path, deleteForce); err != nil {
		return fmt.Errorf("failed to delete worktree: %w", err)
	}

	fmt.Printf("Worktree '%s' deleted successfully!\n", wt.Name())
	return nil
}

package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yunus/wt/internal/git"
	"github.com/yunus/wt/internal/tmux"
	"github.com/yunus/wt/internal/tui"
)

var (
	cleanupDryRun bool
	cleanupForce  bool
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Remove worktrees with deleted remote branches",
	Long: `Find and remove worktrees whose remote branch no longer exists.

This typically happens after a PR is merged and the branch is deleted on GitHub.
An interactive picker lets you select which worktrees to delete.`,
	RunE: runCleanup,
}

func init() {
	cleanupCmd.Flags().BoolVar(&cleanupDryRun, "dry-run", false, "show what would be deleted without deleting")
	cleanupCmd.Flags().BoolVarP(&cleanupForce, "force", "f", false, "delete all stale worktrees without interactive selection")
}

func runCleanup(cmd *cobra.Command, args []string) error {
	repo, err := git.FindRepo()
	if err != nil {
		cwd, _ := os.Getwd()
		return fmt.Errorf("failed to find git repository (cwd: %s): %w", cwd, err)
	}

	fmt.Println("Checking remote branches...")

	staleWorktrees, err := repo.GetStaleWorktrees()
	if err != nil {
		return fmt.Errorf("failed to check worktrees: %w", err)
	}

	if len(staleWorktrees) == 0 {
		fmt.Println("No stale worktrees found. All branches exist on remote.")
		return nil
	}

	currentWt, _ := repo.GetCurrentWorktree()

	// Build picker items
	var items []tui.Item
	for _, wt := range staleWorktrees {
		desc := wt.Path
		if tmux.SessionExists(repo.SessionName(&wt)) {
			desc += " [tmux]"
		}
		if currentWt != nil && wt.Path == currentWt.Path {
			desc += " (current)"
		}
		items = append(items, tui.Item{
			Name:        wt.Name(),
			Description: desc,
			Value:       wt,
		})
	}

	if cleanupDryRun {
		fmt.Printf("\nFound %d worktree(s) with deleted remote branches:\n\n", len(staleWorktrees))
		for _, item := range items {
			fmt.Printf("  - %s  %s\n", item.Name, item.Description)
		}
		fmt.Println("\n(dry-run mode - no changes made)")
		return nil
	}

	// Determine which worktrees to delete
	var toDelete []git.Worktree
	if cleanupForce {
		// Force mode: delete all
		toDelete = staleWorktrees
	} else {
		// Interactive mode: let user select
		selected, err := tui.RunMultiSelectPicker(
			fmt.Sprintf("Select worktrees to delete (%d stale):", len(staleWorktrees)),
			items,
		)
		if err != nil {
			return fmt.Errorf("picker failed: %w", err)
		}
		if selected == nil {
			fmt.Println("Cancelled.")
			return nil
		}
		if len(selected) == 0 {
			fmt.Println("No worktrees selected.")
			return nil
		}

		// Extract Worktree structs from selected items
		for _, item := range selected {
			if wt, ok := item.Value.(git.Worktree); ok {
				toDelete = append(toDelete, wt)
			}
		}
	}

	// Check if current worktree is in the deletion list
	var currentIsStale bool
	for _, wt := range toDelete {
		if currentWt != nil && wt.Path == currentWt.Path {
			currentIsStale = true
			break
		}
	}

	// If we're in a stale worktree, switch to default first
	if currentIsStale && tmux.IsInsideTmux() {
		defaultBranch, _ := repo.GetDefaultBranch()
		defaultWt, err := repo.FindWorktree(defaultBranch)
		if err != nil {
			return fmt.Errorf("cannot find default worktree to switch to: %w", err)
		}

		fmt.Printf("\nSwitching to '%s' before cleanup...\n", defaultWt.Name())
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

	// Delete selected worktrees
	fmt.Println()
	deleted := 0
	for _, wt := range toDelete {
		fmt.Printf("Deleting %s...\n", wt.Name())

		sessionName := repo.SessionName(&wt)

		// Kill tmux session if exists
		if tmux.SessionExists(sessionName) {
			fmt.Printf("  Killing tmux session '%s'...\n", sessionName)
			if err := tmux.KillSession(sessionName); err != nil {
				fmt.Printf("  Warning: failed to kill session: %v\n", err)
			}
		}

		// Delete worktree and branch
		fmt.Printf("  Removing worktree and branch...\n")
		if err := repo.DeleteWorktree(wt.Path, true); err != nil {
			fmt.Printf("  Warning: failed to delete: %v\n", err)
			continue
		}

		deleted++
		fmt.Printf("  Done!\n")
	}

	fmt.Printf("\nCleaned up %d worktree(s).\n", deleted)
	return nil
}

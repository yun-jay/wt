package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yunus/wt/internal/git"
	"github.com/yunus/wt/internal/tmux"
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
The command will show all stale worktrees and ask for confirmation before deleting.`,
	RunE: runCleanup,
}

func init() {
	cleanupCmd.Flags().BoolVar(&cleanupDryRun, "dry-run", false, "show what would be deleted without deleting")
	cleanupCmd.Flags().BoolVarP(&cleanupForce, "force", "f", false, "skip confirmation prompt")
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

	// Check if current worktree is in the stale list
	currentWt, _ := repo.GetCurrentWorktree()
	var currentIsStale bool
	for _, wt := range staleWorktrees {
		if currentWt != nil && wt.Path == currentWt.Path {
			currentIsStale = true
			break
		}
	}

	// Display stale worktrees
	fmt.Printf("\nFound %d worktree(s) with deleted remote branches:\n\n", len(staleWorktrees))
	for i, wt := range staleWorktrees {
		sessionStatus := ""
		if tmux.SessionExists(repo.SessionName(&wt)) {
			sessionStatus = " [tmux]"
		}
		currentMarker := ""
		if currentWt != nil && wt.Path == currentWt.Path {
			currentMarker = " (current)"
		}
		fmt.Printf("  %d. %s%s%s\n", i+1, wt.Name(), sessionStatus, currentMarker)
		fmt.Printf("     Path: %s\n", wt.Path)
	}

	if cleanupDryRun {
		fmt.Println("\n(dry-run mode - no changes made)")
		return nil
	}

	// Warn if current worktree is stale
	if currentIsStale && tmux.IsInsideTmux() {
		fmt.Println("\nWarning: You are currently in a stale worktree.")
		fmt.Println("You will be switched to the default branch before deletion.")
	}

	// Confirm deletion
	if !cleanupForce {
		fmt.Printf("\nDelete all %d worktree(s)? [y/N]: ", len(staleWorktrees))
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled.")
			return nil
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

	// Delete worktrees sequentially
	fmt.Println()
	deleted := 0
	for _, wt := range staleWorktrees {
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

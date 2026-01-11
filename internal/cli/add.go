package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/yunus/wt/internal/files"
	"github.com/yunus/wt/internal/git"
	"github.com/yunus/wt/internal/tmux"
)

var (
	addBase     string
	addNoSwitch bool
)

var addCmd = &cobra.Command{
	Use:   "add <branch>",
	Short: "Create a new worktree",
	Long: `Create a new git worktree and tmux session.

The new worktree is created as a sibling directory to the main worktree.
A tmux session is automatically created with the pane layout from your config.`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

func init() {
	addCmd.Flags().StringVarP(&addBase, "base", "b", "", "base branch (default: auto-detect)")
	addCmd.Flags().BoolVar(&addNoSwitch, "no-switch", false, "don't switch to the new worktree")
}

func runAdd(cmd *cobra.Command, args []string) error {
	branch := args[0]

	repo, err := git.FindRepo()
	if err != nil {
		return fmt.Errorf("failed to find git repository: %w", err)
	}

	// Determine base branch
	baseBranch := addBase
	if baseBranch == "" {
		baseBranch, _ = repo.GetDefaultBranch()
	}

	// Get the default worktree for file operations
	defaultBranch, _ := repo.GetDefaultBranch()
	defaultWt, _ := repo.FindWorktree(defaultBranch)

	// Create worktree
	fmt.Printf("Creating worktree '%s' from '%s'...\n", branch, baseBranch)
	wt, err := repo.CreateWorktree(branch, baseBranch)
	if err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// Setup file symlinks/copies
	if defaultWt != nil && (len(cfg.Files.Symlink) > 0 || len(cfg.Files.Copy) > 0) {
		fmt.Println("Setting up shared files...")
		files.SetupFiles(defaultWt.Path, wt.Path, cfg.Files.Symlink, cfg.Files.Copy)
	}

	// Run setup commands (one-time initialization)
	if len(cfg.Setup) > 0 {
		fmt.Println("Running setup commands...")
		for _, command := range cfg.Setup {
			fmt.Printf("  Running: %s\n", command)
			cmd := exec.Command("bash", "-c", command)
			cmd.Dir = wt.Path
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Printf("  Warning: command failed: %v\n", err)
			}
		}
	}

	// Run post-create commands
	if len(cfg.PostCreate) > 0 {
		fmt.Println("Running post-create commands...")
		for _, command := range cfg.PostCreate {
			fmt.Printf("  Running: %s\n", command)
			cmd := exec.Command("bash", "-c", command)
			cmd.Dir = wt.Path
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Printf("  Warning: command failed: %v\n", err)
			}
		}
	}

	// Create tmux session
	sessionName := repo.SessionName(wt)
	fmt.Printf("Creating tmux session '%s'...\n", sessionName)
	if err := tmux.CreateSession(sessionName, wt.Path, cfg); err != nil {
		// Session might already exist, try to continue
		if err != tmux.ErrSessionExists {
			return fmt.Errorf("failed to create tmux session: %w", err)
		}
	}

	// Switch to the new worktree
	if !addNoSwitch {
		if tmux.IsInsideTmux() {
			fmt.Printf("Switching to session '%s'...\n", sessionName)
		} else {
			fmt.Printf("Attaching to session '%s'...\n", sessionName)
		}
		if err := tmux.SwitchSession(sessionName); err != nil {
			return fmt.Errorf("failed to switch to session: %w", err)
		}
	}

	fmt.Printf("\nWorktree '%s' created successfully!\n", branch)
	return nil
}

package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yunus/wt/internal/config"
	"github.com/yunus/wt/internal/git"
)

var (
	cfg         *config.Config
	cfgFile     string
	projectRoot string // Where .wt.yaml lives (bare repo dir)
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "wt",
	Short: "Git worktree and tmux session manager",
	Long: `wt is a CLI tool that manages git worktrees and tmux sessions.

It provides seamless integration between git worktrees and tmux, allowing you to:
- Create worktrees with automatic tmux session setup
- Switch between worktrees (and their tmux sessions)
- Delete worktrees with session cleanup
- Share files across worktrees via symlinks or copies

Get started:
  wt init              # Create global config
  wt add feature-x     # Create new worktree + session
  wt list              # List all worktrees
  wt switch feature-x  # Switch to worktree`,
	PersistentPreRunE: initializeConfig,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.config/wt/config.yaml)")

	// Add subcommands
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(switchCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(completionsCmd)
	rootCmd.AddCommand(toggleCmd)

	// Bookmark commands
	rootCmd.AddCommand(markCmd)
	rootCmd.AddCommand(unmarkCmd)
	rootCmd.AddCommand(marksCmd)

	// Navigation commands
	rootCmd.AddCommand(nextCmd)
	rootCmd.AddCommand(prevCmd)
	rootCmd.AddCommand(jumpCmd)

	// Indicator commands
	rootCmd.AddCommand(indicatorCmd)
}

func initializeConfig(cmd *cobra.Command, args []string) error {
	// Find project root for project config (bare repo dir)
	if repo, err := git.FindRepo(); err == nil {
		projectRoot = repo.ProjectRoot
	}

	var err error
	cfg, err = config.LoadWithProject(projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
		cfg = config.DefaultConfig()
	}

	return nil
}

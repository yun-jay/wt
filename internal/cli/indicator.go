package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yunus/wt/internal/indicator"
)

var indicatorCmd = &cobra.Command{
	Use:   "indicator",
	Short: "Manage worktree indicators",
	Long: `Manage worktree status indicators.

Indicators are colored symbols displayed next to worktrees in the picker.
External processes can set indicator states via 'wt indicator set'.

Example:
  wt indicator set main claude idle     # Set claude indicator to idle (green)
  wt indicator set main claude loading  # Set claude indicator to loading (orange)
  wt indicator clear main               # Clear all indicators for main
  wt indicator show                     # Show all indicator states`,
}

var indicatorSetCmd = &cobra.Command{
	Use:   "set <worktree> <indicator> <state>",
	Short: "Set indicator state for a worktree",
	Long: `Set the state of an indicator for a worktree.

The state should match one defined in your indicator config (e.g., idle, loading, needs_input).

Example:
  wt indicator set main claude idle
  wt indicator set feature-x claude needs_input`,
	Args: cobra.ExactArgs(3),
	RunE: runIndicatorSet,
}

var indicatorClearCmd = &cobra.Command{
	Use:   "clear <worktree> [indicator]",
	Short: "Clear indicator state",
	Long: `Clear indicator state for a worktree.

If no indicator is specified, clears all indicators for the worktree.

Example:
  wt indicator clear main           # Clear all indicators
  wt indicator clear main claude    # Clear only claude indicator`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runIndicatorClear,
}

var indicatorShowCmd = &cobra.Command{
	Use:   "show [worktree]",
	Short: "Show indicator states",
	Long: `Show current indicator states for worktrees.

If no worktree is specified, shows all worktrees with indicators.

Example:
  wt indicator show                 # Show all
  wt indicator show main            # Show specific worktree`,
	Args: cobra.MaximumNArgs(1),
	RunE: runIndicatorShow,
}

func init() {
	indicatorCmd.AddCommand(indicatorSetCmd)
	indicatorCmd.AddCommand(indicatorClearCmd)
	indicatorCmd.AddCommand(indicatorShowCmd)
}

func getIndicatorManager() *indicator.Manager {
	if cfg == nil || !cfg.HasIndicators() {
		// Use defaults if no config
		return indicator.NewManager("~/.wt/indicators", nil, projectRoot)
	}

	// Convert config definitions to indicator definitions
	defs := make([]indicator.Definition, len(cfg.Indicators.Definitions))
	for i, d := range cfg.Indicators.Definitions {
		defs[i] = indicator.Definition{
			Name:     d.Name,
			Symbols:  d.Symbols,
			Colors:   d.Colors,
			Priority: d.Priority,
		}
	}

	return indicator.NewManager(cfg.GetIndicatorStateDir(), defs, projectRoot)
}

func runIndicatorSet(cmd *cobra.Command, args []string) error {
	worktree := args[0]
	indicatorName := args[1]
	state := args[2]

	mgr := getIndicatorManager()

	if err := mgr.SetIndicator(worktree, indicatorName, state); err != nil {
		return fmt.Errorf("failed to set indicator: %w", err)
	}

	fmt.Printf("Set %s.%s = %s\n", worktree, indicatorName, state)
	return nil
}

func runIndicatorClear(cmd *cobra.Command, args []string) error {
	worktree := args[0]
	indicatorName := ""
	if len(args) > 1 {
		indicatorName = args[1]
	}

	mgr := getIndicatorManager()

	if err := mgr.ClearIndicators(worktree, indicatorName); err != nil {
		return fmt.Errorf("failed to clear indicator: %w", err)
	}

	if indicatorName == "" {
		fmt.Printf("Cleared all indicators for %s\n", worktree)
	} else {
		fmt.Printf("Cleared %s.%s\n", worktree, indicatorName)
	}
	return nil
}

func runIndicatorShow(cmd *cobra.Command, args []string) error {
	mgr := getIndicatorManager()

	if len(args) > 0 {
		// Show specific worktree
		worktree := args[0]
		states, err := mgr.GetAllStates(worktree)
		if err != nil {
			return fmt.Errorf("failed to get states: %w", err)
		}

		if len(states) == 0 {
			fmt.Printf("%s: (no indicators)\n", worktree)
			return nil
		}

		fmt.Printf("%s:\n", worktree)
		for name, state := range states {
			fmt.Printf("  %s: %s\n", name, state)
		}
		return nil
	}

	// Show all worktrees
	worktrees, err := mgr.ListWorktreesWithState()
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	if len(worktrees) == 0 {
		fmt.Println("No indicator states found")
		fmt.Printf("State directory: %s\n", mgr.GetStateDir())
		return nil
	}

	for _, worktree := range worktrees {
		states, err := mgr.GetAllStates(worktree)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to get states for %s: %v\n", worktree, err)
			continue
		}

		if len(states) == 0 {
			continue
		}

		parts := make([]string, 0, len(states))
		for name, state := range states {
			parts = append(parts, fmt.Sprintf("%s=%s", name, state))
		}
		fmt.Printf("%s: %s\n", worktree, strings.Join(parts, ", "))
	}

	return nil
}

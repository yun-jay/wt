package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yunus/wt/internal/git"
	"github.com/yunus/wt/internal/indicator"
	"github.com/yunus/wt/internal/state"
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
		cwd, _ := os.Getwd()
		return fmt.Errorf("failed to find git repository (cwd: %s): %w", cwd, err)
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

		// Load state for sorting by recency
		st, _ := state.LoadState(repo.ProjectRoot)

		// Get worktree names and sort by recency
		wtNames := make([]string, len(worktrees))
		wtMap := make(map[string]git.Worktree)
		for i, wt := range worktrees {
			wtNames[i] = wt.Name()
			wtMap[wt.Name()] = wt
		}
		sortedNames := st.GetSortedWorktrees(wtNames)

		// Create indicator manager if configured
		var indManager *indicator.Manager
		if cfg.HasIndicators() {
			defs := make([]indicator.Definition, len(cfg.Indicators.Definitions))
			for i, d := range cfg.Indicators.Definitions {
				defs[i] = indicator.Definition{
					Name:     d.Name,
					Symbols:  d.Symbols,
					Colors:   d.Colors,
					Priority: d.Priority,
				}
			}
			indManager = indicator.NewManager(cfg.GetIndicatorStateDir(), defs, projectRoot)
		}

		// Build picker items in sorted order
		items := make([]tui.Item, len(sortedNames))
		for i, name := range sortedNames {
			wt := wtMap[name]
			desc := wt.Path
			if tmux.SessionExists(repo.SessionName(&wt)) {
				desc += " [tmux]"
			}
			item := tui.Item{
				Name:         wt.Name(),
				Description:  desc,
				Value:        wt,
				IndicatorKey: wt.Name(),
			}

			// Add indicators if configured
			if indManager != nil {
				results := indManager.GetIndicators(wt.Name())
				for _, r := range results {
					item.Indicators = append(item.Indicators, tui.Indicator{
						Symbol: r.Symbol,
						Color:  r.Color,
					})
				}
			}

			items[i] = item
		}

		// Use picker with live indicator updates if configured
		var selected *tui.Item
		if indManager != nil {
			selected, err = tui.RunPickerWithIndicators("Switch to worktree:", items, indManager.GetStateDir(), func(worktree string) []tui.Indicator {
				results := indManager.GetIndicators(worktree)
				indicators := make([]tui.Indicator, len(results))
				for i, r := range results {
					indicators[i] = tui.Indicator{Symbol: r.Symbol, Color: r.Color}
				}
				return indicators
			})
		} else {
			selected, err = tui.RunPicker("Switch to worktree:", items)
		}
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

	sessionName := repo.SessionName(wt)

	// Record visit in state
	st, _ := state.LoadState(repo.ProjectRoot)
	st.RecordVisit(wt.Name())
	_ = st.Save(repo.ProjectRoot)

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

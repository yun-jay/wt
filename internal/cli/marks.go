package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yunus/wt/internal/config"
	"github.com/yunus/wt/internal/git"
	"github.com/yunus/wt/internal/state"
	"github.com/yunus/wt/internal/tmux"
	"github.com/yunus/wt/internal/tui"
)

var marksCmd = &cobra.Command{
	Use:   "marks",
	Short: "Show and switch to bookmarked worktrees",
	Long: `Display a picker with all bookmarked worktrees.

Press Enter to switch, 'x' to remove from bookmarks.
Bookmarks are ordered by slot number (1, 2, 3, ...).`,
	Args: cobra.NoArgs,
	RunE: runMarks,
}

func runMarks(cmd *cobra.Command, args []string) error {
	repo, err := git.FindRepo()
	if err != nil {
		cwd, _ := os.Getwd()
		return fmt.Errorf("failed to find git repository (cwd: %s): %w", cwd, err)
	}

	// Load project config (create empty if not exists)
	projectCfg, err := config.LoadProject(repo.ProjectRoot)
	if err != nil {
		if os.IsNotExist(err) {
			projectCfg = &config.Config{}
		} else {
			return fmt.Errorf("failed to load project config: %w", err)
		}
	}

	// Get all worktrees to check which bookmarks are valid
	worktrees, err := repo.ListWorktrees()
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Build a map of worktree names to worktrees
	wtMap := make(map[string]*git.Worktree)
	for i := range worktrees {
		wtMap[worktrees[i].Name()] = &worktrees[i]
	}

	// Build picker items from bookmarks
	var items []tui.Item
	var missingBookmarks []string
	for i, bookmarkName := range projectCfg.Bookmarks {
		wt, exists := wtMap[bookmarkName]
		if !exists {
			missingBookmarks = append(missingBookmarks, bookmarkName)
			continue
		}

		desc := wt.Path
		if tmux.SessionExists(wt.SessionName()) {
			desc += " [tmux]"
		}
		items = append(items, tui.Item{
			Name:        fmt.Sprintf("%d. %s", i+1, wt.Name()),
			Description: desc,
			Value:       wt,
		})
	}

	// Warn about missing bookmarks
	for _, missing := range missingBookmarks {
		fmt.Fprintf(os.Stderr, "Warning: bookmark '%s' references missing worktree\n", missing)
	}

	// Run picker with delete support (shows empty message if no bookmarks)
	result, err := tui.RunPickerWithDeleteAndEmpty("Bookmarks:", items, func(item tui.Item) error {
		wt := item.Value.(*git.Worktree)
		projectCfg.RemoveBookmark(wt.Name())
		return projectCfg.SaveProject(repo.ProjectRoot)
	}, "No bookmarks. Use 'wt mark' to add the current worktree.")
	if err != nil {
		return fmt.Errorf("picker failed: %w", err)
	}
	if result == nil {
		return nil // User cancelled or no selection
	}

	// Switch to selected worktree
	wt := result.Value.(*git.Worktree)
	sessionName := wt.SessionName()

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

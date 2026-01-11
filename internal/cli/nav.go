package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/yunus/wt/internal/config"
	"github.com/yunus/wt/internal/git"
	"github.com/yunus/wt/internal/state"
	"github.com/yunus/wt/internal/tmux"
)

var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "Switch to next bookmark",
	Long:  `Switch to the next bookmark in the list. Wraps around to the first bookmark.`,
	Args:  cobra.NoArgs,
	RunE:  runNext,
}

var prevCmd = &cobra.Command{
	Use:   "prev",
	Short: "Switch to previous bookmark",
	Long:  `Switch to the previous bookmark in the list. Wraps around to the last bookmark.`,
	Args:  cobra.NoArgs,
	RunE:  runPrev,
}

var jumpCmd = &cobra.Command{
	Use:   "jump <n>",
	Short: "Jump to bookmark at index",
	Long:  `Jump directly to the bookmark at the given index (1-based).`,
	Args:  cobra.ExactArgs(1),
	RunE:  runJump,
}

func runNext(cmd *cobra.Command, args []string) error {
	return navigateBookmark(1)
}

func runPrev(cmd *cobra.Command, args []string) error {
	return navigateBookmark(-1)
}

func runJump(cmd *cobra.Command, args []string) error {
	index, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid bookmark index: %s", args[0])
	}

	if index < 1 {
		return fmt.Errorf("bookmark index must be >= 1")
	}

	repo, err := git.FindRepo()
	if err != nil {
		return fmt.Errorf("failed to find git repository: %w", err)
	}

	projectCfg, err := config.LoadProject(repo.ProjectRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no bookmarks configured")
		}
		return fmt.Errorf("failed to load project config: %w", err)
	}

	if len(projectCfg.Bookmarks) == 0 {
		return fmt.Errorf("no bookmarks configured")
	}

	if index > len(projectCfg.Bookmarks) {
		return fmt.Errorf("bookmark index %d out of range (%d bookmarks)", index, len(projectCfg.Bookmarks))
	}

	targetName := projectCfg.Bookmarks[index-1]
	return switchToWorktree(repo, targetName)
}

func navigateBookmark(direction int) error {
	repo, err := git.FindRepo()
	if err != nil {
		return fmt.Errorf("failed to find git repository: %w", err)
	}

	projectCfg, err := config.LoadProject(repo.ProjectRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no bookmarks configured")
		}
		return fmt.Errorf("failed to load project config: %w", err)
	}

	if len(projectCfg.Bookmarks) == 0 {
		return fmt.Errorf("no bookmarks configured")
	}

	// With only 1 bookmark, do nothing
	if len(projectCfg.Bookmarks) == 1 {
		return nil
	}

	// Find current worktree
	currentWt, err := repo.GetCurrentWorktree()
	var currentIdx int
	if err != nil {
		// Not in a worktree, start from first or last based on direction
		if direction > 0 {
			currentIdx = -1
		} else {
			currentIdx = len(projectCfg.Bookmarks)
		}
	} else {
		currentIdx = projectCfg.GetBookmarkIndex(currentWt.Name())
		if currentIdx == -1 {
			// Current worktree not bookmarked, start from first or last
			if direction > 0 {
				currentIdx = -1
			} else {
				currentIdx = len(projectCfg.Bookmarks)
			}
		}
	}

	// Calculate next index with wrapping
	nextIdx := currentIdx + direction
	if nextIdx < 0 {
		nextIdx = len(projectCfg.Bookmarks) - 1
	} else if nextIdx >= len(projectCfg.Bookmarks) {
		nextIdx = 0
	}

	targetName := projectCfg.Bookmarks[nextIdx]
	return switchToWorktree(repo, targetName)
}

func switchToWorktree(repo *git.Repo, wtName string) error {
	wt, err := repo.FindWorktree(wtName)
	if err != nil {
		return fmt.Errorf("worktree '%s' not found", wtName)
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

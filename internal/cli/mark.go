package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yunus/wt/internal/config"
	"github.com/yunus/wt/internal/git"
)

var markCmd = &cobra.Command{
	Use:   "mark",
	Short: "Add current worktree to bookmarks",
	Long:  `Add the current worktree to the bookmark list for quick navigation.`,
	Args:  cobra.NoArgs,
	RunE:  runMark,
}

var unmarkCmd = &cobra.Command{
	Use:   "unmark",
	Short: "Remove current worktree from bookmarks",
	Long:  `Remove the current worktree from the bookmark list.`,
	Args:  cobra.NoArgs,
	RunE:  runUnmark,
}

func runMark(cmd *cobra.Command, args []string) error {
	repo, err := git.FindRepo()
	if err != nil {
		return fmt.Errorf("failed to find git repository: %w", err)
	}

	wt, err := repo.GetCurrentWorktree()
	if err != nil {
		return fmt.Errorf("not inside a worktree")
	}

	// Load existing project config or create new one
	projectCfg, err := config.LoadProject(repo.ProjectRoot)
	if err != nil {
		if os.IsNotExist(err) {
			projectCfg = &config.Config{}
		} else {
			return fmt.Errorf("failed to load project config: %w", err)
		}
	}

	wtName := wt.Name()
	if projectCfg.IsBookmarked(wtName) {
		fmt.Printf("'%s' is already bookmarked\n", wtName)
		return nil
	}

	projectCfg.AddBookmark(wtName)

	if err := projectCfg.SaveProject(repo.ProjectRoot); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	idx := projectCfg.GetBookmarkIndex(wtName) + 1
	fmt.Printf("Bookmarked '%s' at slot %d\n", wtName, idx)
	return nil
}

func runUnmark(cmd *cobra.Command, args []string) error {
	repo, err := git.FindRepo()
	if err != nil {
		return fmt.Errorf("failed to find git repository: %w", err)
	}

	wt, err := repo.GetCurrentWorktree()
	if err != nil {
		return fmt.Errorf("not inside a worktree")
	}

	// Load existing project config
	projectCfg, err := config.LoadProject(repo.ProjectRoot)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("'%s' is not bookmarked\n", wt.Name())
			return nil
		}
		return fmt.Errorf("failed to load project config: %w", err)
	}

	wtName := wt.Name()
	if !projectCfg.IsBookmarked(wtName) {
		fmt.Printf("'%s' is not bookmarked\n", wtName)
		return nil
	}

	projectCfg.RemoveBookmark(wtName)

	if err := projectCfg.SaveProject(repo.ProjectRoot); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Removed '%s' from bookmarks\n", wtName)
	return nil
}

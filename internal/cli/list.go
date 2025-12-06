package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/yunus/wt/internal/git"
	"github.com/yunus/wt/internal/tmux"
)

var (
	listJSON  bool
	listQuiet bool
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all worktrees",
	Long:    `List all git worktrees for the current repository.`,
	RunE:    runList,
}

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "output as JSON")
	listCmd.Flags().BoolVarP(&listQuiet, "quiet", "q", false, "only output worktree names")
}

type worktreeOutput struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Branch     string `json:"branch"`
	IsCurrent  bool   `json:"is_current"`
	HasSession bool   `json:"has_session"`
}

func runList(cmd *cobra.Command, args []string) error {
	repo, err := git.FindRepo()
	if err != nil {
		return fmt.Errorf("failed to find git repository: %w", err)
	}

	worktrees, err := repo.ListWorktrees()
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Get current worktree
	currentWt, _ := repo.GetCurrentWorktree()

	// Build output data
	var outputs []worktreeOutput
	for _, wt := range worktrees {
		isCurrent := currentWt != nil && wt.Path == currentWt.Path
		hasSession := tmux.SessionExists(wt.SessionName())
		outputs = append(outputs, worktreeOutput{
			Name:       wt.Name(),
			Path:       wt.Path,
			Branch:     wt.Branch,
			IsCurrent:  isCurrent,
			HasSession: hasSession,
		})
	}

	// Output based on format
	if listJSON {
		return outputJSON(outputs)
	}

	if listQuiet {
		return outputQuiet(outputs)
	}

	return outputTable(outputs)
}

func outputJSON(outputs []worktreeOutput) error {
	data, err := json.MarshalIndent(outputs, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func outputQuiet(outputs []worktreeOutput) error {
	for _, o := range outputs {
		fmt.Println(o.Name)
	}
	return nil
}

func outputTable(outputs []worktreeOutput) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	for _, o := range outputs {
		markers := []string{}
		if o.IsCurrent {
			markers = append(markers, "*")
		}
		if o.HasSession {
			markers = append(markers, "tmux")
		}

		marker := ""
		if len(markers) > 0 {
			marker = "[" + strings.Join(markers, ",") + "]"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\n", o.Name, o.Path, marker)
	}

	return nil
}

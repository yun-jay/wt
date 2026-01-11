package git

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	ErrWorktreeNotFound = errors.New("worktree not found")
	ErrWorktreeExists   = errors.New("worktree already exists")
	ErrBranchExists     = errors.New("branch already exists")
)

// Worktree represents a git worktree
type Worktree struct {
	Path   string
	Branch string
	Commit string
	IsMain bool // Is this the main/master/dev worktree?
	IsBare bool // Is this a bare worktree entry?
}

// Name returns the worktree name (directory basename)
func (w *Worktree) Name() string {
	return filepath.Base(w.Path)
}

// ListWorktrees returns all worktrees for the repository
func (r *Repo) ListWorktrees() ([]Worktree, error) {
	cmd := exec.Command("git", "--git-dir", r.GitDir, "worktree", "list", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	var worktrees []Worktree
	var current *Worktree

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "worktree ") {
			if current != nil {
				worktrees = append(worktrees, *current)
			}
			current = &Worktree{
				Path: strings.TrimPrefix(line, "worktree "),
			}
		} else if strings.HasPrefix(line, "HEAD ") {
			if current != nil {
				current.Commit = strings.TrimPrefix(line, "HEAD ")
			}
		} else if strings.HasPrefix(line, "branch ") {
			if current != nil {
				// Branch is like "refs/heads/main"
				branch := strings.TrimPrefix(line, "branch ")
				current.Branch = strings.TrimPrefix(branch, "refs/heads/")
			}
		} else if line == "bare" {
			if current != nil {
				current.IsBare = true
			}
		}
	}

	// Don't forget the last one
	if current != nil {
		worktrees = append(worktrees, *current)
	}

	// Filter out bare entries and mark main branches
	var result []Worktree
	defaultBranch, _ := r.GetDefaultBranch()
	for _, wt := range worktrees {
		if wt.IsBare {
			continue
		}
		if wt.Branch == defaultBranch || wt.Branch == "main" || wt.Branch == "master" || wt.Branch == "dev" {
			wt.IsMain = true
		}
		result = append(result, wt)
	}

	return result, nil
}

// FindWorktree finds a worktree by name or path
func (r *Repo) FindWorktree(nameOrPath string) (*Worktree, error) {
	worktrees, err := r.ListWorktrees()
	if err != nil {
		return nil, err
	}

	for _, wt := range worktrees {
		if wt.Name() == nameOrPath || wt.Path == nameOrPath || wt.Branch == nameOrPath {
			return &wt, nil
		}
	}

	return nil, ErrWorktreeNotFound
}

// CreateWorktree creates a new worktree as a sibling directory
func (r *Repo) CreateWorktree(branch, baseBranch string) (*Worktree, error) {
	// Check if branch already exists
	cmd := exec.Command("git", "--git-dir", r.GitDir, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	branchExists := cmd.Run() == nil

	// Determine the worktree path (inside the bare repo directory)
	// e.g., foo.git + branch "main" -> foo.git/main
	wtPath := filepath.Join(r.GitDir, branch)

	// Check if path already exists
	if _, err := os.Stat(wtPath); err == nil {
		return nil, ErrWorktreeExists
	}

	var createCmd *exec.Cmd
	if branchExists {
		// Branch exists, just create worktree for it
		createCmd = exec.Command("git", "--git-dir", r.GitDir, "worktree", "add", wtPath, branch)
	} else {
		// Create new branch from base
		if baseBranch == "" {
			baseBranch, _ = r.GetDefaultBranch()
		}
		createCmd = exec.Command("git", "--git-dir", r.GitDir, "worktree", "add", "-b", branch, wtPath, baseBranch)
	}

	createCmd.Stdout = os.Stdout
	createCmd.Stderr = os.Stderr
	if err := createCmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to create worktree: %w", err)
	}

	return &Worktree{
		Path:   wtPath,
		Branch: branch,
	}, nil
}

// DeleteWorktree removes a worktree
func (r *Repo) DeleteWorktree(nameOrPath string, force bool) error {
	wt, err := r.FindWorktree(nameOrPath)
	if err != nil {
		return err
	}

	// Remove the worktree
	args := []string{"--git-dir", r.GitDir, "worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, wt.Path)

	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Try force if normal removal fails
		if !force {
			args = []string{"--git-dir", r.GitDir, "worktree", "remove", "--force", wt.Path}
			cmd = exec.Command("git", args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to remove worktree: %w", err)
			}
		} else {
			return fmt.Errorf("failed to remove worktree: %w", err)
		}
	}

	// Optionally delete the branch (only if not a protected branch)
	if !wt.IsMain {
		deleteCmd := exec.Command("git", "--git-dir", r.GitDir, "branch", "-D", wt.Branch)
		_ = deleteCmd.Run() // Ignore errors, branch might be needed elsewhere
	}

	return nil
}

// GetWorktreePath returns the path where a new worktree with the given branch would be created
func (r *Repo) GetWorktreePath(branch string) string {
	return filepath.Join(r.GitDir, branch)
}

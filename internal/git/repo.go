package git

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	ErrNotGitRepo  = errors.New("not a git repository")
	ErrNotBareRepo = errors.New("not a bare repository")
	ErrNoWorktrees = errors.New("no worktrees found")
)

// Repo represents a bare git repository
type Repo struct {
	// GitDir is the path to the .git directory (bare repo)
	GitDir string
	// WorktreeRoot is the parent directory where worktrees are created
	WorktreeRoot string
	// ProjectRoot is where the project config (.wt.yaml) should be stored
	// For bare repos, this is the GitDir itself (e.g., project.git/)
	ProjectRoot string
}

// FindRepo finds the bare git repository from the current directory
// It expects a bare repo structure where worktrees are sibling directories
func FindRepo() (*Repo, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return FindRepoFrom(cwd)
}

// FindRepoFrom finds the bare git repository starting from the given path
func FindRepoFrom(path string) (*Repo, error) {
	// First, check if we're inside a worktree or git directory
	cmd := exec.Command("git", "-C", path, "rev-parse", "--git-common-dir")
	output, err := cmd.Output()
	if err != nil {
		return nil, ErrNotGitRepo
	}

	gitCommonDir := strings.TrimSpace(string(output))

	// Make the path absolute if it's relative
	if !filepath.IsAbs(gitCommonDir) {
		gitCommonDir = filepath.Join(path, gitCommonDir)
	}
	gitCommonDir = filepath.Clean(gitCommonDir)

	// Check if this is a bare repo by checking git config
	isBare, err := isBareRepo(gitCommonDir)
	if err != nil || !isBare {
		return nil, ErrNotBareRepo
	}

	// For bare repos, gitCommonDir is the bare repo directory (e.g., project.git)
	return &Repo{
		GitDir:       gitCommonDir,
		WorktreeRoot: filepath.Dir(gitCommonDir),
		ProjectRoot:  gitCommonDir, // Config lives in the bare repo dir
	}, nil
}

// ProjectName returns the project name (bare repo dir without .git suffix)
func (r *Repo) ProjectName() string {
	base := filepath.Base(r.GitDir)
	return strings.TrimSuffix(base, ".git")
}

// SessionName returns the tmux session name for a worktree
// Format: {project}_{worktree} e.g., "myproject_feat1"
// This ensures unique session names across different repositories
func (r *Repo) SessionName(wt *Worktree) string {
	base := r.ProjectName() + "_" + wt.Name()
	return strings.ReplaceAll(base, ".", "_")
}

// isBareRepo checks if the git directory is a bare repository
func isBareRepo(gitDir string) (bool, error) {
	cmd := exec.Command("git", "--git-dir", gitDir, "config", "--get", "core.bare")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(output)) == "true", nil
}

// GetDefaultBranch returns the default branch name (main, master, or dev)
func (r *Repo) GetDefaultBranch() (string, error) {
	// Try git symbolic-ref first
	cmd := exec.Command("git", "--git-dir", r.GitDir, "symbolic-ref", "refs/remotes/origin/HEAD")
	if output, err := cmd.Output(); err == nil {
		// Output is like "refs/remotes/origin/main"
		ref := strings.TrimSpace(string(output))
		parts := strings.Split(ref, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1], nil
		}
	}

	// Check if branches exist in refs
	for _, branch := range []string{"main", "master", "dev"} {
		cmd := exec.Command("git", "--git-dir", r.GitDir, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
		if cmd.Run() == nil {
			return branch, nil
		}
	}

	return "main", nil // Default fallback
}

// GetCurrentWorktree returns the worktree for the current directory
func (r *Repo) GetCurrentWorktree() (*Worktree, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// Resolve symlinks for accurate comparison (e.g., /tmp -> /private/tmp on macOS)
	cwdReal, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		cwdReal = cwd
	}

	worktrees, err := r.ListWorktrees()
	if err != nil {
		return nil, err
	}

	for _, wt := range worktrees {
		wtPathReal, err := filepath.EvalSymlinks(wt.Path)
		if err != nil {
			wtPathReal = wt.Path
		}
		if strings.HasPrefix(cwdReal, wtPathReal) || strings.HasPrefix(cwd, wt.Path) {
			return &wt, nil
		}
	}

	return nil, errors.New("not inside a worktree")
}

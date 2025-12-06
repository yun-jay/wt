package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func setupTestBareRepo(t *testing.T) (string, func()) {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "wt-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create bare repo
	bareRepoPath := filepath.Join(tmpDir, "test-project.git")
	cmd := exec.Command("git", "init", "--bare", bareRepoPath)
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create bare repo: %v", err)
	}

	// Create initial commit in a temp worktree
	initWorktree := filepath.Join(tmpDir, "init-wt")
	cmd = exec.Command("git", "clone", bareRepoPath, initWorktree)
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to clone bare repo: %v", err)
	}

	// Create initial commit
	testFile := filepath.Join(initWorktree, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test"), 0644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "-C", initWorktree, "add", ".")
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to git add: %v", err)
	}

	cmd = exec.Command("git", "-C", initWorktree, "commit", "-m", "Initial commit")
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@test.com", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@test.com")
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to git commit: %v", err)
	}

	cmd = exec.Command("git", "-C", initWorktree, "push", "origin", "main")
	if err := cmd.Run(); err != nil {
		// Try master instead
		cmd = exec.Command("git", "-C", initWorktree, "push", "origin", "master")
		if err := cmd.Run(); err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to git push: %v", err)
		}
	}

	// Remove the init worktree
	os.RemoveAll(initWorktree)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return bareRepoPath, cleanup
}

func TestFindRepoFrom(t *testing.T) {
	bareRepoPath, cleanup := setupTestBareRepo(t)
	defer cleanup()

	// Test finding repo from bare repo directory
	repo, err := FindRepoFrom(bareRepoPath)
	if err != nil {
		t.Fatalf("FindRepoFrom failed: %v", err)
	}

	if repo.GitDir != bareRepoPath {
		t.Errorf("GitDir = %s, want %s", repo.GitDir, bareRepoPath)
	}

	if repo.ProjectRoot != bareRepoPath {
		t.Errorf("ProjectRoot = %s, want %s", repo.ProjectRoot, bareRepoPath)
	}
}

func TestFindRepoFromNonBareRepo(t *testing.T) {
	// Create temp directory with regular git repo
	tmpDir, err := os.MkdirTemp("", "wt-test-regular-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cmd := exec.Command("git", "init", tmpDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init regular repo: %v", err)
	}

	// Should return ErrNotBareRepo
	_, err = FindRepoFrom(tmpDir)
	if err != ErrNotBareRepo {
		t.Errorf("FindRepoFrom = %v, want ErrNotBareRepo", err)
	}
}

func TestProjectName(t *testing.T) {
	tests := []struct {
		gitDir   string
		expected string
	}{
		{"/repos/foo.git", "foo"},
		{"/repos/my-project.git", "my-project"},
		{"/repos/test.git", "test"},
	}

	for _, tt := range tests {
		repo := &Repo{GitDir: tt.gitDir}
		got := repo.ProjectName()
		if got != tt.expected {
			t.Errorf("ProjectName() for %s = %s, want %s", tt.gitDir, got, tt.expected)
		}
	}
}

func TestIsBareRepo(t *testing.T) {
	bareRepoPath, cleanup := setupTestBareRepo(t)
	defer cleanup()

	isBare, err := isBareRepo(bareRepoPath)
	if err != nil {
		t.Fatalf("isBareRepo failed: %v", err)
	}

	if !isBare {
		t.Error("isBareRepo = false, want true")
	}
}

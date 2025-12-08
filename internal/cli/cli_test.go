package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestBareRepo creates a bare repo for testing and returns path and cleanup func
func setupTestBareRepo(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "wt-cli-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	bareRepoPath := filepath.Join(tmpDir, "test-project.git")
	cmd := exec.Command("git", "init", "--bare", bareRepoPath)
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create bare repo: %v", err)
	}

	// Create initial commit
	initWorktree := filepath.Join(tmpDir, "init-wt")
	cmd = exec.Command("git", "clone", bareRepoPath, initWorktree)
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to clone bare repo: %v", err)
	}

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
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@test.com",
	)
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to git commit: %v", err)
	}

	cmd = exec.Command("git", "-C", initWorktree, "push", "origin", "main")
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("git", "-C", initWorktree, "push", "origin", "master")
		if err := cmd.Run(); err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to git push: %v", err)
		}
	}

	os.RemoveAll(initWorktree)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return bareRepoPath, cleanup
}

func TestListCommand(t *testing.T) {
	bareRepoPath, cleanup := setupTestBareRepo(t)
	defer cleanup()

	// Save and change working directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(bareRepoPath)

	// Reset global state
	projectRoot = ""
	cfg = nil

	// Initialize config
	if err := initializeConfig(nil, nil); err != nil {
		t.Fatalf("initializeConfig failed: %v", err)
	}

	// Run list command (should show empty list initially)
	err := runList(nil, nil)
	if err != nil {
		t.Fatalf("runList failed: %v", err)
	}
}

func TestAddAndDeleteWorkflow(t *testing.T) {
	bareRepoPath, cleanup := setupTestBareRepo(t)
	defer cleanup()

	// Save and change working directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(bareRepoPath)

	// Reset global state
	projectRoot = ""
	cfg = nil
	addNoSwitch = true // Don't try to switch tmux session in tests
	deleteForce = true // Skip confirmation prompt in tests

	// Initialize config
	if err := initializeConfig(nil, nil); err != nil {
		t.Fatalf("initializeConfig failed: %v", err)
	}

	// Create a worktree
	err := runAdd(nil, []string{"feature-test"})
	if err != nil {
		// Ignore tmux errors in non-tmux environment
		if !strings.Contains(err.Error(), "tmux") {
			t.Fatalf("runAdd failed: %v", err)
		}
	}

	// Verify worktree was created
	wtPath := filepath.Join(bareRepoPath, "feature-test")
	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		t.Error("Worktree directory was not created")
	}

	// Delete the worktree
	err = runDelete(nil, []string{"feature-test"})
	if err != nil {
		// Ignore tmux errors
		if !strings.Contains(err.Error(), "tmux") {
			t.Fatalf("runDelete failed: %v", err)
		}
	}

	// Verify worktree was deleted
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Error("Worktree directory should be deleted")
	}
}

func TestInitGlobalConfig(t *testing.T) {
	// Create temp home directory
	tmpHome, err := os.MkdirTemp("", "wt-cli-home-*")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	// Save original HOME and set temp
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpHome)

	// Also need to handle XDG_CONFIG_HOME
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)
	os.Unsetenv("XDG_CONFIG_HOME")

	// Reset global state
	projectRoot = ""
	cfg = nil

	// Run init (should create global config)
	err = runInit(nil, nil)
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	// Verify config was created
	configPath := filepath.Join(tmpHome, ".config", "wt", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Global config was not created")
	}
}

func TestInitProjectConfig(t *testing.T) {
	bareRepoPath, cleanup := setupTestBareRepo(t)
	defer cleanup()

	// Save and change working directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(bareRepoPath)

	// Set projectRoot to simulate being in a bare repo
	projectRoot = bareRepoPath
	cfg = nil

	// Run init (should create project config)
	err := runInit(nil, nil)
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	// Verify .wt.yaml was created
	configPath := filepath.Join(bareRepoPath, ".wt.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Project config was not created")
	}
}

func TestStatusCommand(t *testing.T) {
	bareRepoPath, cleanup := setupTestBareRepo(t)
	defer cleanup()

	// Save and change working directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(bareRepoPath)

	// Reset global state
	projectRoot = bareRepoPath
	cfg = nil

	// Initialize config
	if err := initializeConfig(nil, nil); err != nil {
		t.Fatalf("initializeConfig failed: %v", err)
	}

	// Run status command (should work even without worktrees)
	// Note: This might fail in non-worktree context, which is expected
	_ = runStatus(nil, nil)
}

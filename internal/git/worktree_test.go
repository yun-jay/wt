package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateWorktree(t *testing.T) {
	bareRepoPath, cleanup := setupTestBareRepo(t)
	defer cleanup()

	repo, err := FindRepoFrom(bareRepoPath)
	if err != nil {
		t.Fatalf("FindRepoFrom failed: %v", err)
	}

	// Create a worktree
	wt, err := repo.CreateWorktree("test-branch", "")
	if err != nil {
		t.Fatalf("CreateWorktree failed: %v", err)
	}

	// Verify worktree path is inside bare repo
	expectedPath := filepath.Join(bareRepoPath, "test-branch")
	if wt.Path != expectedPath {
		t.Errorf("Worktree path = %s, want %s", wt.Path, expectedPath)
	}

	// Verify worktree directory exists
	if _, err := os.Stat(wt.Path); os.IsNotExist(err) {
		t.Error("Worktree directory was not created")
	}

	// Verify branch name
	if wt.Branch != "test-branch" {
		t.Errorf("Branch = %s, want test-branch", wt.Branch)
	}
}

func TestCreateWorktreeInsideBareRepo(t *testing.T) {
	bareRepoPath, cleanup := setupTestBareRepo(t)
	defer cleanup()

	repo, err := FindRepoFrom(bareRepoPath)
	if err != nil {
		t.Fatalf("FindRepoFrom failed: %v", err)
	}

	// Create multiple worktrees
	branches := []string{"feature-a", "feature-b", "bugfix"}
	for _, branch := range branches {
		wt, err := repo.CreateWorktree(branch, "")
		if err != nil {
			t.Fatalf("CreateWorktree(%s) failed: %v", branch, err)
		}

		// All worktrees should be inside the bare repo
		if filepath.Dir(wt.Path) != bareRepoPath {
			t.Errorf("Worktree %s not inside bare repo: %s", branch, wt.Path)
		}
	}

	// List worktrees and verify
	worktrees, err := repo.ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees failed: %v", err)
	}

	// Should have 3 worktrees (excluding bare entry)
	if len(worktrees) != 3 {
		t.Errorf("ListWorktrees count = %d, want 3", len(worktrees))
	}
}

func TestListWorktrees(t *testing.T) {
	bareRepoPath, cleanup := setupTestBareRepo(t)
	defer cleanup()

	repo, err := FindRepoFrom(bareRepoPath)
	if err != nil {
		t.Fatalf("FindRepoFrom failed: %v", err)
	}

	// Initially should have no worktrees (bare entry is filtered out)
	worktrees, err := repo.ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees failed: %v", err)
	}

	if len(worktrees) != 0 {
		t.Errorf("Initial worktree count = %d, want 0", len(worktrees))
	}

	// Create a worktree
	_, err = repo.CreateWorktree("main", "")
	if err != nil {
		t.Fatalf("CreateWorktree failed: %v", err)
	}

	// Now should have 1 worktree
	worktrees, err = repo.ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees failed: %v", err)
	}

	if len(worktrees) != 1 {
		t.Errorf("Worktree count = %d, want 1", len(worktrees))
	}
}

func TestFindWorktree(t *testing.T) {
	bareRepoPath, cleanup := setupTestBareRepo(t)
	defer cleanup()

	repo, err := FindRepoFrom(bareRepoPath)
	if err != nil {
		t.Fatalf("FindRepoFrom failed: %v", err)
	}

	// Create a worktree
	created, err := repo.CreateWorktree("feature-test", "")
	if err != nil {
		t.Fatalf("CreateWorktree failed: %v", err)
	}

	// Find by name
	found, err := repo.FindWorktree("feature-test")
	if err != nil {
		t.Fatalf("FindWorktree by name failed: %v", err)
	}

	// Compare paths after resolving symlinks (macOS /tmp -> /private/tmp)
	createdReal, _ := filepath.EvalSymlinks(created.Path)
	foundReal, _ := filepath.EvalSymlinks(found.Path)
	if foundReal != createdReal {
		t.Errorf("FindWorktree path = %s, want %s", found.Path, created.Path)
	}

	// Find by branch
	found, err = repo.FindWorktree("feature-test")
	if err != nil {
		t.Fatalf("FindWorktree by branch failed: %v", err)
	}

	if found.Branch != "feature-test" {
		t.Errorf("FindWorktree branch = %s, want feature-test", found.Branch)
	}

	// Find non-existent worktree
	_, err = repo.FindWorktree("non-existent")
	if err != ErrWorktreeNotFound {
		t.Errorf("FindWorktree for non-existent = %v, want ErrWorktreeNotFound", err)
	}
}

func TestDeleteWorktree(t *testing.T) {
	bareRepoPath, cleanup := setupTestBareRepo(t)
	defer cleanup()

	repo, err := FindRepoFrom(bareRepoPath)
	if err != nil {
		t.Fatalf("FindRepoFrom failed: %v", err)
	}

	// Create a worktree
	wt, err := repo.CreateWorktree("to-delete", "")
	if err != nil {
		t.Fatalf("CreateWorktree failed: %v", err)
	}

	// Delete it
	err = repo.DeleteWorktree("to-delete", false)
	if err != nil {
		t.Fatalf("DeleteWorktree failed: %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(wt.Path); !os.IsNotExist(err) {
		t.Error("Worktree directory still exists after deletion")
	}

	// Verify it's not in the list
	_, err = repo.FindWorktree("to-delete")
	if err != ErrWorktreeNotFound {
		t.Error("Worktree still findable after deletion")
	}
}

func TestGetWorktreePath(t *testing.T) {
	repo := &Repo{
		GitDir:       "/repos/foo.git",
		WorktreeRoot: "/repos",
		ProjectRoot:  "/repos/foo.git",
	}

	path := repo.GetWorktreePath("feature-x")
	expected := "/repos/foo.git/feature-x"

	if path != expected {
		t.Errorf("GetWorktreePath = %s, want %s", path, expected)
	}
}

func TestRepoSessionName(t *testing.T) {
	repo := &Repo{
		GitDir:       "/repos/foo.git",
		WorktreeRoot: "/repos",
		ProjectRoot:  "/repos/foo.git",
	}

	wt := &Worktree{
		Path:   "/repos/foo.git/feature.branch",
		Branch: "feature.branch",
	}

	// Session name should include project name and replace dots with underscores
	sessionName := repo.SessionName(wt)
	expected := "foo_feature_branch"

	if sessionName != expected {
		t.Errorf("SessionName = %s, want %s", sessionName, expected)
	}
}

func TestRepoSessionNameUniqueness(t *testing.T) {
	// Two different repos with same branch name should have different session names
	repoA := &Repo{
		GitDir:       "/repos/project-a.git",
		WorktreeRoot: "/repos",
		ProjectRoot:  "/repos/project-a.git",
	}

	repoB := &Repo{
		GitDir:       "/repos/project-b.git",
		WorktreeRoot: "/repos",
		ProjectRoot:  "/repos/project-b.git",
	}

	wt := &Worktree{
		Path:   "feat1", // Same worktree name
		Branch: "feat1",
	}

	sessionA := repoA.SessionName(wt)
	sessionB := repoB.SessionName(wt)

	if sessionA == sessionB {
		t.Errorf("Session names should be unique: repo-a=%s, repo-b=%s", sessionA, sessionB)
	}

	expectedA := "project-a_feat1"
	expectedB := "project-b_feat1"

	if sessionA != expectedA {
		t.Errorf("Session name for repo-a = %s, want %s", sessionA, expectedA)
	}

	if sessionB != expectedB {
		t.Errorf("Session name for repo-b = %s, want %s", sessionB, expectedB)
	}
}

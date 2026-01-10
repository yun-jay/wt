package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadState_NewFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wt-state-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	state, err := LoadState(tmpDir)
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if state == nil {
		t.Fatal("LoadState returned nil")
	}

	if len(state.History) != 0 {
		t.Errorf("History length = %d, want 0", len(state.History))
	}
}

func TestLoadState_ExistingFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wt-state-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a state file
	stateContent := `{"history": ["main", "feature-x"], "last_updated": "2024-01-01T00:00:00Z"}`
	statePath := filepath.Join(tmpDir, StateFile)
	if err := os.WriteFile(statePath, []byte(stateContent), 0644); err != nil {
		t.Fatalf("failed to write state file: %v", err)
	}

	state, err := LoadState(tmpDir)
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if len(state.History) != 2 {
		t.Errorf("History length = %d, want 2", len(state.History))
	}

	if state.History[0] != "main" {
		t.Errorf("History[0] = %s, want main", state.History[0])
	}
}

func TestRecordVisit_AddsToFront(t *testing.T) {
	state := &State{
		History: []string{"main", "feature-x"},
	}

	state.RecordVisit("bugfix-y")

	if len(state.History) != 3 {
		t.Errorf("History length = %d, want 3", len(state.History))
	}

	if state.History[0] != "bugfix-y" {
		t.Errorf("History[0] = %s, want bugfix-y", state.History[0])
	}
}

func TestRecordVisit_Deduplicates(t *testing.T) {
	state := &State{
		History: []string{"main", "feature-x", "bugfix-y"},
	}

	state.RecordVisit("feature-x")

	if len(state.History) != 3 {
		t.Errorf("History length = %d, want 3", len(state.History))
	}

	if state.History[0] != "feature-x" {
		t.Errorf("History[0] = %s, want feature-x", state.History[0])
	}

	// feature-x should not appear twice
	count := 0
	for _, h := range state.History {
		if h == "feature-x" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("feature-x appears %d times, want 1", count)
	}
}

func TestRecordVisit_LimitsTo20(t *testing.T) {
	state := &State{
		History: make([]string, 20),
	}
	for i := 0; i < 20; i++ {
		state.History[i] = "old"
	}

	state.RecordVisit("new")

	if len(state.History) != MaxHistory {
		t.Errorf("History length = %d, want %d", len(state.History), MaxHistory)
	}

	if state.History[0] != "new" {
		t.Errorf("History[0] = %s, want new", state.History[0])
	}
}

func TestGetSortedWorktrees(t *testing.T) {
	state := &State{
		History: []string{"feature-x", "main", "bugfix-y"},
	}

	all := []string{"main", "feature-x", "new-branch", "bugfix-y"}
	sorted := state.GetSortedWorktrees(all)

	// Should be: feature-x (0), main (1), bugfix-y (2), new-branch (unknown, at end)
	expected := []string{"feature-x", "main", "bugfix-y", "new-branch"}

	if len(sorted) != len(expected) {
		t.Fatalf("sorted length = %d, want %d", len(sorted), len(expected))
	}

	for i, want := range expected {
		if sorted[i] != want {
			t.Errorf("sorted[%d] = %s, want %s", i, sorted[i], want)
		}
	}
}

func TestGetSortedWorktrees_EmptyHistory(t *testing.T) {
	state := &State{
		History: []string{},
	}

	all := []string{"main", "feature-x", "bugfix-y"}
	sorted := state.GetSortedWorktrees(all)

	// With empty history, should maintain original order
	if len(sorted) != 3 {
		t.Fatalf("sorted length = %d, want 3", len(sorted))
	}

	for i, want := range all {
		if sorted[i] != want {
			t.Errorf("sorted[%d] = %s, want %s", i, sorted[i], want)
		}
	}
}

func TestSave_WritesJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wt-state-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	state := &State{
		History: []string{"main", "feature-x"},
	}

	if err := state.Save(tmpDir); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	statePath := filepath.Join(tmpDir, StateFile)
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Error("State file was not created")
	}

	// Load and verify
	loaded, err := LoadState(tmpDir)
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if len(loaded.History) != 2 {
		t.Errorf("Loaded history length = %d, want 2", len(loaded.History))
	}

	if loaded.History[0] != "main" {
		t.Errorf("Loaded History[0] = %s, want main", loaded.History[0])
	}
}

func TestSave_UpdatesLastUpdated(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wt-state-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	state := &State{
		History: []string{},
	}

	originalTime := state.LastUpdated

	if err := state.Save(tmpDir); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if !state.LastUpdated.After(originalTime) {
		t.Error("LastUpdated was not updated on save")
	}
}

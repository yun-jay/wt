package indicator

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "indicator-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func testDefinitions() []Definition {
	return []Definition{
		{
			Name: "claude",
			Symbols: map[string]string{
				"idle":        "●",
				"loading":     "●",
				"needs_input": "●",
				"_default":    "○",
			},
			Colors: map[string]string{
				"idle":        "42",
				"loading":     "214",
				"needs_input": "196",
				"_default":    "240",
			},
			Priority: 1,
		},
		{
			Name: "build",
			Symbols: map[string]string{
				"passing":  "✓",
				"failing":  "✗",
				"_default": "○",
			},
			Colors: map[string]string{
				"passing":  "42",
				"failing":  "196",
				"_default": "240",
			},
			Priority: 2,
		},
	}
}

func TestNewManager(t *testing.T) {
	dir := setupTestDir(t)
	defs := testDefinitions()

	m := NewManager(dir, defs)

	if m.stateDir != dir {
		t.Errorf("expected stateDir %s, got %s", dir, m.stateDir)
	}
	if len(m.definitions) != 2 {
		t.Errorf("expected 2 definitions, got %d", len(m.definitions))
	}
}

func TestGetIndicators_NoFile(t *testing.T) {
	dir := setupTestDir(t)
	m := NewManager(dir, testDefinitions())

	results := m.GetIndicators("main")

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Should use defaults when no file exists
	if results[0].Symbol != "○" {
		t.Errorf("expected default symbol ○, got %s", results[0].Symbol)
	}
	if results[0].Color != "240" {
		t.Errorf("expected default color 240, got %s", results[0].Color)
	}
}

func TestGetIndicators_ValidFile(t *testing.T) {
	dir := setupTestDir(t)
	m := NewManager(dir, testDefinitions())

	// Create a state file
	if err := m.SetIndicator("main", "claude", "idle"); err != nil {
		t.Fatalf("failed to set indicator: %v", err)
	}

	results := m.GetIndicators("main")

	// Claude should be idle (green)
	var claude *Result
	for i := range results {
		if results[i].Name == "claude" {
			claude = &results[i]
			break
		}
	}

	if claude == nil {
		t.Fatal("claude indicator not found")
	}
	if claude.State != "idle" {
		t.Errorf("expected state idle, got %s", claude.State)
	}
	if claude.Symbol != "●" {
		t.Errorf("expected symbol ●, got %s", claude.Symbol)
	}
	if claude.Color != "42" {
		t.Errorf("expected color 42, got %s", claude.Color)
	}
}

func TestGetIndicators_InvalidJSON(t *testing.T) {
	dir := setupTestDir(t)
	m := NewManager(dir, testDefinitions())

	// Write invalid JSON
	os.MkdirAll(dir, 0755)
	if err := os.WriteFile(filepath.Join(dir, "main.json"), []byte("invalid json"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	results := m.GetIndicators("main")

	// Should return defaults on parse error
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Symbol != "○" {
		t.Errorf("expected default symbol ○, got %s", results[0].Symbol)
	}
}

func TestGetIndicators_UnknownState(t *testing.T) {
	dir := setupTestDir(t)
	m := NewManager(dir, testDefinitions())

	// Set an unknown state
	if err := m.SetIndicator("main", "claude", "unknown_state"); err != nil {
		t.Fatalf("failed to set indicator: %v", err)
	}

	results := m.GetIndicators("main")

	var claude *Result
	for i := range results {
		if results[i].Name == "claude" {
			claude = &results[i]
			break
		}
	}

	if claude == nil {
		t.Fatal("claude indicator not found")
	}
	// Should use _default for unknown state
	if claude.Symbol != "○" {
		t.Errorf("expected default symbol ○, got %s", claude.Symbol)
	}
	if claude.Color != "240" {
		t.Errorf("expected default color 240, got %s", claude.Color)
	}
}

func TestGetIndicators_Priority(t *testing.T) {
	dir := setupTestDir(t)

	// Swap priorities to test sorting
	defs := []Definition{
		{Name: "second", Priority: 2, Symbols: map[string]string{"_default": "2"}, Colors: map[string]string{"_default": "2"}},
		{Name: "first", Priority: 1, Symbols: map[string]string{"_default": "1"}, Colors: map[string]string{"_default": "1"}},
		{Name: "third", Priority: 3, Symbols: map[string]string{"_default": "3"}, Colors: map[string]string{"_default": "3"}},
	}

	m := NewManager(dir, defs)
	results := m.GetIndicators("main")

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Should be sorted by priority
	if results[0].Name != "first" {
		t.Errorf("expected first indicator to be 'first', got %s", results[0].Name)
	}
	if results[1].Name != "second" {
		t.Errorf("expected second indicator to be 'second', got %s", results[1].Name)
	}
	if results[2].Name != "third" {
		t.Errorf("expected third indicator to be 'third', got %s", results[2].Name)
	}
}

func TestSetIndicator_CreatesFile(t *testing.T) {
	dir := setupTestDir(t)
	m := NewManager(dir, testDefinitions())

	if err := m.SetIndicator("main", "claude", "loading"); err != nil {
		t.Fatalf("failed to set indicator: %v", err)
	}

	// File should exist
	path := filepath.Join(dir, "main.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected state file to be created")
	}

	// Check content
	states, err := m.GetAllStates("main")
	if err != nil {
		t.Fatalf("failed to get states: %v", err)
	}
	if states["claude"] != "loading" {
		t.Errorf("expected state loading, got %s", states["claude"])
	}
}

func TestSetIndicator_UpdatesFile(t *testing.T) {
	dir := setupTestDir(t)
	m := NewManager(dir, testDefinitions())

	// Set initial state
	if err := m.SetIndicator("main", "claude", "idle"); err != nil {
		t.Fatalf("failed to set indicator: %v", err)
	}

	// Add another indicator
	if err := m.SetIndicator("main", "build", "passing"); err != nil {
		t.Fatalf("failed to set indicator: %v", err)
	}

	states, err := m.GetAllStates("main")
	if err != nil {
		t.Fatalf("failed to get states: %v", err)
	}

	if states["claude"] != "idle" {
		t.Errorf("expected claude=idle, got %s", states["claude"])
	}
	if states["build"] != "passing" {
		t.Errorf("expected build=passing, got %s", states["build"])
	}
}

func TestClearIndicators_All(t *testing.T) {
	dir := setupTestDir(t)
	m := NewManager(dir, testDefinitions())

	// Set some indicators
	m.SetIndicator("main", "claude", "idle")
	m.SetIndicator("main", "build", "passing")

	// Clear all
	if err := m.ClearIndicators("main", ""); err != nil {
		t.Fatalf("failed to clear indicators: %v", err)
	}

	// File should be gone
	path := filepath.Join(dir, "main.json")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected state file to be removed")
	}
}

func TestClearIndicators_Specific(t *testing.T) {
	dir := setupTestDir(t)
	m := NewManager(dir, testDefinitions())

	// Set some indicators
	m.SetIndicator("main", "claude", "idle")
	m.SetIndicator("main", "build", "passing")

	// Clear specific
	if err := m.ClearIndicators("main", "claude"); err != nil {
		t.Fatalf("failed to clear indicator: %v", err)
	}

	states, err := m.GetAllStates("main")
	if err != nil {
		t.Fatalf("failed to get states: %v", err)
	}

	if _, ok := states["claude"]; ok {
		t.Error("expected claude to be cleared")
	}
	if states["build"] != "passing" {
		t.Errorf("expected build=passing, got %s", states["build"])
	}
}

func TestResolveSymbol(t *testing.T) {
	def := Definition{
		Symbols: map[string]string{
			"idle":     "●",
			"_default": "○",
		},
	}

	if sym := def.resolveSymbol("idle"); sym != "●" {
		t.Errorf("expected ●, got %s", sym)
	}
	if sym := def.resolveSymbol("unknown"); sym != "○" {
		t.Errorf("expected ○ for unknown, got %s", sym)
	}
}

func TestResolveColor(t *testing.T) {
	def := Definition{
		Colors: map[string]string{
			"idle":     "42",
			"_default": "240",
		},
	}

	if col := def.resolveColor("idle"); col != "42" {
		t.Errorf("expected 42, got %s", col)
	}
	if col := def.resolveColor("unknown"); col != "240" {
		t.Errorf("expected 240 for unknown, got %s", col)
	}
}

func TestListWorktreesWithState(t *testing.T) {
	dir := setupTestDir(t)
	m := NewManager(dir, testDefinitions())

	// No files initially
	worktrees, err := m.ListWorktreesWithState()
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(worktrees) != 0 {
		t.Errorf("expected 0 worktrees, got %d", len(worktrees))
	}

	// Add some state files
	m.SetIndicator("main", "claude", "idle")
	m.SetIndicator("feature-x", "claude", "loading")

	worktrees, err = m.ListWorktreesWithState()
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(worktrees) != 2 {
		t.Errorf("expected 2 worktrees, got %d", len(worktrees))
	}
}

func TestStateFilePath_Sanitization(t *testing.T) {
	dir := setupTestDir(t)
	m := NewManager(dir, testDefinitions())

	// Worktree names with special chars should be sanitized
	path := m.stateFilePath("feature/branch")
	expected := filepath.Join(dir, "feature_branch.json")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestGetStateDir(t *testing.T) {
	dir := setupTestDir(t)
	m := NewManager(dir, testDefinitions())

	if m.GetStateDir() != dir {
		t.Errorf("expected %s, got %s", dir, m.GetStateDir())
	}
}

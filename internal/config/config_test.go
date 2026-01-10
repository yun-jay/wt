package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	if len(cfg.Windows) == 0 {
		t.Error("DefaultConfig has no windows")
	}

	if cfg.TogglePane == nil {
		t.Error("DefaultConfig has no toggle pane")
	}

	if cfg.Agent == "" {
		t.Error("DefaultConfig has no agent")
	}

	// Verify default protected branches
	if len(cfg.ProtectedBranches) != 3 {
		t.Errorf("ProtectedBranches count = %d, want 3", len(cfg.ProtectedBranches))
	}

	// Verify toggle pane defaults
	if cfg.TogglePane.Position != "right" {
		t.Errorf("TogglePane.Position = %s, want right", cfg.TogglePane.Position)
	}
	if cfg.TogglePane.Size != 20 {
		t.Errorf("TogglePane.Size = %d, want 20", cfg.TogglePane.Size)
	}
}

func TestHasTogglePane(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *Config
		expected bool
	}{
		{
			name:     "with toggle pane",
			cfg:      &Config{TogglePane: &TogglePaneConfig{Name: "claude"}},
			expected: true,
		},
		{
			name:     "without toggle pane",
			cfg:      &Config{TogglePane: nil},
			expected: false,
		},
		{
			name:     "empty toggle pane name",
			cfg:      &Config{TogglePane: &TogglePaneConfig{Name: ""}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.HasTogglePane()
			if got != tt.expected {
				t.Errorf("HasTogglePane() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestExpandAgentPlaceholder(t *testing.T) {
	cfg := &Config{Agent: "claude"}

	tests := []struct {
		input    string
		expected string
	}{
		{"<agent>", "claude"},
		{"run <agent> --flag", "run claude --flag"},
		{"no placeholder", "no placeholder"},
		{"<agent> <agent>", "claude claude"},
	}

	for _, tt := range tests {
		got := cfg.ExpandAgentPlaceholder(tt.input)
		if got != tt.expected {
			t.Errorf("ExpandAgentPlaceholder(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestIsProtectedBranch(t *testing.T) {
	cfg := &Config{
		ProtectedBranches: []string{"main", "master", "dev"},
	}

	tests := []struct {
		branch   string
		expected bool
	}{
		{"main", true},
		{"master", true},
		{"dev", true},
		{"feature-x", false},
		{"Main", false}, // case sensitive
	}

	for _, tt := range tests {
		got := cfg.IsProtectedBranch(tt.branch)
		if got != tt.expected {
			t.Errorf("IsProtectedBranch(%q) = %v, want %v", tt.branch, got, tt.expected)
		}
	}
}

func TestSaveAndLoadProject(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wt-config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &Config{
		Windows: []Window{
			{Name: "test-window", Focus: true},
		},
		Agent: "test-agent",
	}

	// Save
	if err := cfg.SaveProject(tmpDir); err != nil {
		t.Fatalf("SaveProject failed: %v", err)
	}

	// Verify file exists
	configPath := filepath.Join(tmpDir, ProjectConfigFile)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Load
	loaded, err := LoadProject(tmpDir)
	if err != nil {
		t.Fatalf("LoadProject failed: %v", err)
	}

	if len(loaded.Windows) != 1 {
		t.Errorf("Loaded windows count = %d, want 1", len(loaded.Windows))
	}

	if loaded.Windows[0].Name != "test-window" {
		t.Errorf("Loaded window name = %s, want test-window", loaded.Windows[0].Name)
	}

	if loaded.Agent != "test-agent" {
		t.Errorf("Loaded agent = %s, want test-agent", loaded.Agent)
	}
}

func TestMergeConfigs(t *testing.T) {
	global := &Config{
		Agent:             "global-agent",
		ProtectedBranches: []string{"main"},
		PostCreate:        []string{"npm install"},
		Windows: []Window{
			{Name: "global-window"},
		},
	}

	project := &Config{
		Agent: "project-agent",
		Windows: []Window{
			{Name: "project-window"},
		},
		PostCreate: []string{"<global>", "custom-command"},
	}

	merged := mergeConfigs(global, project)

	// Agent should be overridden
	if merged.Agent != "project-agent" {
		t.Errorf("Agent = %s, want project-agent", merged.Agent)
	}

	// Windows should be overridden
	if len(merged.Windows) != 1 || merged.Windows[0].Name != "project-window" {
		t.Error("Windows not properly overridden")
	}

	// PostCreate should merge with <global> expansion
	if len(merged.PostCreate) != 2 {
		t.Errorf("PostCreate count = %d, want 2", len(merged.PostCreate))
	}

	if merged.PostCreate[0] != "npm install" {
		t.Errorf("PostCreate[0] = %s, want npm install", merged.PostCreate[0])
	}

	if merged.PostCreate[1] != "custom-command" {
		t.Errorf("PostCreate[1] = %s, want custom-command", merged.PostCreate[1])
	}
}

func TestProjectConfigExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wt-config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Should not exist initially
	if ProjectConfigExists(tmpDir) {
		t.Error("ProjectConfigExists = true, want false (no config yet)")
	}

	// Create config
	cfg := DefaultConfig()
	if err := cfg.SaveProject(tmpDir); err != nil {
		t.Fatalf("SaveProject failed: %v", err)
	}

	// Should exist now
	if !ProjectConfigExists(tmpDir) {
		t.Error("ProjectConfigExists = false, want true (after save)")
	}
}

func TestGetAgentCommand(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *Config
		expected string
	}{
		{
			name:     "with agent configured",
			cfg:      &Config{Agent: "custom-agent"},
			expected: "custom-agent",
		},
		{
			name:     "without agent configured",
			cfg:      &Config{Agent: ""},
			expected: "claude",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.GetAgentCommand()
			if got != tt.expected {
				t.Errorf("GetAgentCommand() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestGetTogglePaneCommand(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *Config
		expected string
	}{
		{
			name: "with toggle pane and agent placeholder",
			cfg: &Config{
				Agent: "claude",
				TogglePane: &TogglePaneConfig{
					Command: "<agent> --interactive",
				},
			},
			expected: "claude --interactive",
		},
		{
			name: "with toggle pane without placeholder",
			cfg: &Config{
				Agent: "claude",
				TogglePane: &TogglePaneConfig{
					Command: "htop",
				},
			},
			expected: "htop",
		},
		{
			name:     "without toggle pane",
			cfg:      &Config{TogglePane: nil},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.GetTogglePaneCommand()
			if got != tt.expected {
				t.Errorf("GetTogglePaneCommand() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestExpandAgentPlaceholderEmpty(t *testing.T) {
	cfg := &Config{Agent: ""}

	// When agent is empty, placeholder should not be replaced
	input := "<agent> command"
	got := cfg.ExpandAgentPlaceholder(input)
	if got != input {
		t.Errorf("ExpandAgentPlaceholder with empty agent = %s, want %s", got, input)
	}
}

func TestMergeStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		global   []string
		project  []string
		expected []string
	}{
		{
			name:     "empty project uses global",
			global:   []string{"a", "b"},
			project:  []string{},
			expected: []string{"a", "b"},
		},
		{
			name:     "project without global marker replaces",
			global:   []string{"a", "b"},
			project:  []string{"x", "y"},
			expected: []string{"x", "y"},
		},
		{
			name:     "project with global marker at start",
			global:   []string{"a", "b"},
			project:  []string{"<global>", "x"},
			expected: []string{"a", "b", "x"},
		},
		{
			name:     "project with global marker in middle",
			global:   []string{"a", "b"},
			project:  []string{"x", "<global>", "y"},
			expected: []string{"x", "a", "b", "y"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeStringSlice(tt.global, tt.project)
			if len(got) != len(tt.expected) {
				t.Errorf("mergeStringSlice() length = %d, want %d", len(got), len(tt.expected))
				return
			}
			for i, v := range got {
				if v != tt.expected[i] {
					t.Errorf("mergeStringSlice()[%d] = %s, want %s", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestSaveAndLoadGlobal(t *testing.T) {
	// Create temp home directory
	tmpHome, err := os.MkdirTemp("", "wt-config-home-*")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	// Save original HOME and set temp
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpHome)

	// Also handle XDG_CONFIG_HOME
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)
	os.Unsetenv("XDG_CONFIG_HOME")

	cfg := &Config{
		Agent: "test-global-agent",
		Windows: []Window{
			{Name: "test-window"},
		},
	}

	// Save global config
	if err := cfg.SaveGlobal(); err != nil {
		t.Fatalf("SaveGlobal failed: %v", err)
	}

	// Load global config
	loaded, err := LoadGlobal()
	if err != nil {
		t.Fatalf("LoadGlobal failed: %v", err)
	}

	if loaded.Agent != "test-global-agent" {
		t.Errorf("Loaded agent = %s, want test-global-agent", loaded.Agent)
	}
}

func TestLoadWithProject(t *testing.T) {
	// Create temp directories
	tmpHome, err := os.MkdirTemp("", "wt-config-home-*")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	tmpProject, err := os.MkdirTemp("", "wt-config-project-*")
	if err != nil {
		t.Fatalf("failed to create temp project: %v", err)
	}
	defer os.RemoveAll(tmpProject)

	// Save original HOME and set temp
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpHome)

	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)
	os.Unsetenv("XDG_CONFIG_HOME")

	// Create global config
	globalCfg := &Config{
		Agent:      "global-agent",
		PostCreate: []string{"global-cmd"},
	}
	if err := globalCfg.SaveGlobal(); err != nil {
		t.Fatalf("SaveGlobal failed: %v", err)
	}

	// Create project config
	projectCfg := &Config{
		Agent:      "project-agent",
		PostCreate: []string{"<global>", "project-cmd"},
	}
	if err := projectCfg.SaveProject(tmpProject); err != nil {
		t.Fatalf("SaveProject failed: %v", err)
	}

	// Load merged config
	merged, err := LoadWithProject(tmpProject)
	if err != nil {
		t.Fatalf("LoadWithProject failed: %v", err)
	}

	// Project agent should override
	if merged.Agent != "project-agent" {
		t.Errorf("Merged agent = %s, want project-agent", merged.Agent)
	}

	// PostCreate should have both
	if len(merged.PostCreate) != 2 {
		t.Errorf("PostCreate count = %d, want 2", len(merged.PostCreate))
	}
}

func TestMergeConfigsFiles(t *testing.T) {
	global := &Config{
		Files: Files{
			Symlink: []string{".env"},
			Copy:    []string{"config.json"},
		},
	}

	project := &Config{
		Files: Files{
			Symlink: []string{"<global>", ".secrets"},
			Copy:    []string{"settings.yaml"},
		},
	}

	merged := mergeConfigs(global, project)

	// Symlinks should merge
	if len(merged.Files.Symlink) != 2 {
		t.Errorf("Symlink count = %d, want 2", len(merged.Files.Symlink))
	}

	// Copy should be replaced (no <global>)
	if len(merged.Files.Copy) != 1 {
		t.Errorf("Copy count = %d, want 1", len(merged.Files.Copy))
	}
	if merged.Files.Copy[0] != "settings.yaml" {
		t.Errorf("Copy[0] = %s, want settings.yaml", merged.Files.Copy[0])
	}
}

func TestMergeConfigsTogglePane(t *testing.T) {
	global := &Config{
		TogglePane: &TogglePaneConfig{
			Name:     "global-pane",
			Position: "left",
		},
	}

	project := &Config{
		TogglePane: &TogglePaneConfig{
			Name:     "project-pane",
			Position: "right",
		},
	}

	merged := mergeConfigs(global, project)

	// Project toggle pane should override
	if merged.TogglePane.Name != "project-pane" {
		t.Errorf("TogglePane.Name = %s, want project-pane", merged.TogglePane.Name)
	}
	if merged.TogglePane.Position != "right" {
		t.Errorf("TogglePane.Position = %s, want right", merged.TogglePane.Position)
	}
}

func TestMergeConfigsWindowPrefix(t *testing.T) {
	global := &Config{
		WindowPrefix: "global-",
	}

	project := &Config{
		WindowPrefix: "project-",
	}

	merged := mergeConfigs(global, project)

	if merged.WindowPrefix != "project-" {
		t.Errorf("WindowPrefix = %s, want project-", merged.WindowPrefix)
	}
}

func TestEnsureGlobalConfigDir(t *testing.T) {
	// Create temp home directory
	tmpHome, err := os.MkdirTemp("", "wt-config-home-*")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	// Save original HOME and set temp
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpHome)

	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)
	os.Unsetenv("XDG_CONFIG_HOME")

	// Ensure config dir
	if err := EnsureGlobalConfigDir(); err != nil {
		t.Fatalf("EnsureGlobalConfigDir failed: %v", err)
	}

	// Verify directory exists
	configDir := filepath.Join(tmpHome, ".config", "wt")
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Error("Config directory was not created")
	}
}

func TestAddBookmark(t *testing.T) {
	cfg := &Config{Bookmarks: []string{}}

	// Add first bookmark
	if !cfg.AddBookmark("main") {
		t.Error("AddBookmark returned false for new bookmark")
	}

	if len(cfg.Bookmarks) != 1 {
		t.Errorf("Bookmarks length = %d, want 1", len(cfg.Bookmarks))
	}

	if cfg.Bookmarks[0] != "main" {
		t.Errorf("Bookmarks[0] = %s, want main", cfg.Bookmarks[0])
	}

	// Add second bookmark
	cfg.AddBookmark("feature-x")
	if len(cfg.Bookmarks) != 2 {
		t.Errorf("Bookmarks length = %d, want 2", len(cfg.Bookmarks))
	}
}

func TestAddBookmark_NoDuplicates(t *testing.T) {
	cfg := &Config{Bookmarks: []string{"main"}}

	// Try to add duplicate
	if cfg.AddBookmark("main") {
		t.Error("AddBookmark returned true for duplicate")
	}

	if len(cfg.Bookmarks) != 1 {
		t.Errorf("Bookmarks length = %d, want 1", len(cfg.Bookmarks))
	}
}

func TestRemoveBookmark(t *testing.T) {
	cfg := &Config{Bookmarks: []string{"main", "feature-x", "bugfix-y"}}

	// Remove middle bookmark
	if !cfg.RemoveBookmark("feature-x") {
		t.Error("RemoveBookmark returned false for existing bookmark")
	}

	if len(cfg.Bookmarks) != 2 {
		t.Errorf("Bookmarks length = %d, want 2", len(cfg.Bookmarks))
	}

	// Verify order is preserved
	if cfg.Bookmarks[0] != "main" {
		t.Errorf("Bookmarks[0] = %s, want main", cfg.Bookmarks[0])
	}
	if cfg.Bookmarks[1] != "bugfix-y" {
		t.Errorf("Bookmarks[1] = %s, want bugfix-y", cfg.Bookmarks[1])
	}
}

func TestRemoveBookmark_NotFound(t *testing.T) {
	cfg := &Config{Bookmarks: []string{"main"}}

	// Try to remove non-existent bookmark
	if cfg.RemoveBookmark("nonexistent") {
		t.Error("RemoveBookmark returned true for non-existent bookmark")
	}

	if len(cfg.Bookmarks) != 1 {
		t.Errorf("Bookmarks length = %d, want 1", len(cfg.Bookmarks))
	}
}

func TestIsBookmarked(t *testing.T) {
	cfg := &Config{Bookmarks: []string{"main", "feature-x"}}

	tests := []struct {
		name     string
		expected bool
	}{
		{"main", true},
		{"feature-x", true},
		{"nonexistent", false},
	}

	for _, tt := range tests {
		got := cfg.IsBookmarked(tt.name)
		if got != tt.expected {
			t.Errorf("IsBookmarked(%q) = %v, want %v", tt.name, got, tt.expected)
		}
	}
}

func TestGetBookmarkIndex(t *testing.T) {
	cfg := &Config{Bookmarks: []string{"main", "feature-x", "bugfix-y"}}

	tests := []struct {
		name     string
		expected int
	}{
		{"main", 0},
		{"feature-x", 1},
		{"bugfix-y", 2},
		{"nonexistent", -1},
	}

	for _, tt := range tests {
		got := cfg.GetBookmarkIndex(tt.name)
		if got != tt.expected {
			t.Errorf("GetBookmarkIndex(%q) = %d, want %d", tt.name, got, tt.expected)
		}
	}
}

func TestBookmarksSaveAndLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wt-config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &Config{
		Bookmarks: []string{"main", "feature-x"},
	}

	// Save
	if err := cfg.SaveProject(tmpDir); err != nil {
		t.Fatalf("SaveProject failed: %v", err)
	}

	// Load
	loaded, err := LoadProject(tmpDir)
	if err != nil {
		t.Fatalf("LoadProject failed: %v", err)
	}

	if len(loaded.Bookmarks) != 2 {
		t.Errorf("Loaded Bookmarks length = %d, want 2", len(loaded.Bookmarks))
	}

	if loaded.Bookmarks[0] != "main" {
		t.Errorf("Loaded Bookmarks[0] = %s, want main", loaded.Bookmarks[0])
	}
}

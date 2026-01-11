package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	GlobalConfigDir   = ".config/wt"
	GlobalConfigFile  = "config.yaml"
	ProjectConfigFile = ".wt.yaml"
)

// Config represents the merged configuration (global + project)
type Config struct {
	// WindowPrefix is prepended to tmux window names
	WindowPrefix string `yaml:"window_prefix,omitempty"`

	// Windows defines the tmux windows (tabs) for worktree sessions
	Windows []Window `yaml:"windows,omitempty"`

	// TogglePane defines a pane that can be toggled in any window
	TogglePane *TogglePaneConfig `yaml:"toggle_pane,omitempty"`

	// Setup commands to run once when creating a new worktree
	Setup []string `yaml:"setup,omitempty"`

	// PostCreate commands to run after creating worktrees
	PostCreate []string `yaml:"post_create,omitempty"`

	// Files configuration for symlinks and copies
	Files Files `yaml:"files,omitempty"`

	// Agent to use for <agent> placeholder in commands
	Agent string `yaml:"agent,omitempty"`

	// ProtectedBranches that cannot be deleted
	ProtectedBranches []string `yaml:"protected_branches,omitempty"`

	// Bookmarks is an ordered list of bookmarked worktree names
	Bookmarks []string `yaml:"bookmarks,omitempty"`

	// Indicators configures the worktree status indicator system
	Indicators *IndicatorConfig `yaml:"indicators,omitempty"`
}

// IndicatorConfig configures the indicator system
type IndicatorConfig struct {
	// StateDir is where indicator state files are stored (default: ~/.wt/indicators)
	StateDir string `yaml:"state_dir,omitempty"`

	// Definitions defines how each indicator type is displayed
	Definitions []IndicatorDefinition `yaml:"definitions,omitempty"`
}

// IndicatorDefinition describes how to display a particular indicator type
type IndicatorDefinition struct {
	// Name is the indicator identifier (e.g., "claude", "build")
	Name string `yaml:"name"`

	// Symbols maps state values to display symbols
	Symbols map[string]string `yaml:"symbols"`

	// Colors maps state values to lipgloss color codes
	Colors map[string]string `yaml:"colors"`

	// Priority determines display order (lower = shown first)
	Priority int `yaml:"priority,omitempty"`
}

// Window represents a tmux window (tab)
type Window struct {
	Name  string `yaml:"name"`
	Panes []Pane `yaml:"panes,omitempty"`
	Focus bool   `yaml:"focus,omitempty"` // Focus this window on startup
}

// TogglePaneConfig defines a pane that can be toggled in any window
type TogglePaneConfig struct {
	Name     string    `yaml:"name"`
	Command  string    `yaml:"command,omitempty"`
	Position string    `yaml:"position,omitempty"` // "left" or "right"
	Size     int       `yaml:"size,omitempty"`     // Percentage
	Hooks    PaneHooks `yaml:"hooks,omitempty"`
}

// Pane represents a tmux pane within a window
type Pane struct {
	Name    string `yaml:"name,omitempty"`    // Identifier for the pane
	Command string `yaml:"command,omitempty"` // Command to run
	Split   string `yaml:"split,omitempty"`   // "horizontal" or "vertical"
	Size    int    `yaml:"size,omitempty"`    // Percentage size
}

// PaneHooks defines hooks for pane lifecycle
type PaneHooks struct {
	OnOpen  string `yaml:"on_open,omitempty"`  // Command to run when pane is shown
	OnClose string `yaml:"on_close,omitempty"` // Command to run when pane is hidden
}

// Files configuration for sharing across worktrees
type Files struct {
	Symlink []string `yaml:"symlink,omitempty"`
	Copy    []string `yaml:"copy,omitempty"`
}

// ClaudeConfig holds Claude-specific settings
type ClaudeConfig struct {
	Command  string `yaml:"command,omitempty"`
	PaneSize int    `yaml:"pane_size,omitempty"`
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		WindowPrefix: "",
		Windows: []Window{
			{
				Name:  "nvim",
				Focus: true,
				Panes: []Pane{
					{Name: "editor", Command: "nvim ."},
				},
			},
			{
				Name: "shell",
				Panes: []Pane{
					{Name: "shell"},
				},
			},
		},
		TogglePane: &TogglePaneConfig{
			Name:     "claude",
			Command:  "claude",
			Position: "right",
			Size:     20,
		},
		PostCreate:        []string{},
		Agent:             "claude",
		ProtectedBranches: []string{"main", "master", "dev"},
	}
}

// GlobalConfigDir returns the global configuration directory path
func GlobalConfigDirPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, GlobalConfigDir)
}

// GlobalConfigPath returns the global config file path
func GlobalConfigPath() string {
	return filepath.Join(GlobalConfigDirPath(), GlobalConfigFile)
}

// Load loads and merges global and project configs
func Load() (*Config, error) {
	return LoadWithProject("")
}

// LoadWithProject loads and merges global config with project config from given directory
func LoadWithProject(projectDir string) (*Config, error) {
	// Start with defaults
	cfg := DefaultConfig()

	// Load global config
	globalPath := GlobalConfigPath()
	if data, err := os.ReadFile(globalPath); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	// Load project config if directory specified
	if projectDir != "" {
		projectPath := filepath.Join(projectDir, ProjectConfigFile)
		if data, err := os.ReadFile(projectPath); err == nil {
			projectCfg := &Config{}
			if err := yaml.Unmarshal(data, projectCfg); err != nil {
				return nil, err
			}
			cfg = mergeConfigs(cfg, projectCfg)
		}
	}

	return cfg, nil
}

// LoadGlobal loads only the global config
func LoadGlobal() (*Config, error) {
	cfg := DefaultConfig()
	globalPath := GlobalConfigPath()

	data, err := os.ReadFile(globalPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// LoadProject loads only the project config from current directory
func LoadProject(dir string) (*Config, error) {
	projectPath := filepath.Join(dir, ProjectConfigFile)

	data, err := os.ReadFile(projectPath)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// SaveGlobal saves the config as global config
func (c *Config) SaveGlobal() error {
	configDir := GlobalConfigDirPath()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(GlobalConfigPath(), data, 0644)
}

// SaveProject saves the config as project config in the given directory
func (c *Config) SaveProject(dir string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, ProjectConfigFile), data, 0644)
}

// ProjectConfigExists checks if a project config exists in the given directory
func ProjectConfigExists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ProjectConfigFile))
	return err == nil
}

// IsProtectedBranch checks if a branch is protected
func (c *Config) IsProtectedBranch(branch string) bool {
	for _, b := range c.ProtectedBranches {
		if b == branch {
			return true
		}
	}
	return false
}

// EnsureGlobalConfigDir creates the global config directory if it doesn't exist
func EnsureGlobalConfigDir() error {
	return os.MkdirAll(GlobalConfigDirPath(), 0755)
}

// mergeConfigs merges project config into global config
// Project config values override global, except for special <global> markers
func mergeConfigs(global, project *Config) *Config {
	merged := *global

	// Override simple values
	if project.WindowPrefix != "" {
		merged.WindowPrefix = project.WindowPrefix
	}
	if project.Agent != "" {
		merged.Agent = project.Agent
	}

	// Merge windows (project completely overrides if present)
	if len(project.Windows) > 0 {
		merged.Windows = project.Windows
	}

	// Merge toggle pane (project overrides if present)
	if project.TogglePane != nil {
		merged.TogglePane = project.TogglePane
	}

	// Merge setup (handle <global> marker)
	merged.Setup = mergeStringSlice(global.Setup, project.Setup)

	// Merge post_create (handle <global> marker)
	merged.PostCreate = mergeStringSlice(global.PostCreate, project.PostCreate)

	// Merge files
	merged.Files.Symlink = mergeStringSlice(global.Files.Symlink, project.Files.Symlink)
	merged.Files.Copy = mergeStringSlice(global.Files.Copy, project.Files.Copy)

	// Merge protected branches
	if len(project.ProtectedBranches) > 0 {
		merged.ProtectedBranches = project.ProtectedBranches
	}

	// Merge indicators (project overrides if present)
	if project.Indicators != nil {
		merged.Indicators = project.Indicators
	}

	return &merged
}

// mergeStringSlice handles merging with <global> marker support
func mergeStringSlice(global, project []string) []string {
	if len(project) == 0 {
		return global
	}

	result := []string{}
	for _, item := range project {
		if strings.TrimSpace(item) == "<global>" {
			result = append(result, global...)
		} else {
			result = append(result, item)
		}
	}
	return result
}

// ExpandAgentPlaceholder replaces <agent> with the configured agent command
func (c *Config) ExpandAgentPlaceholder(command string) string {
	if c.Agent == "" {
		return command
	}
	return strings.ReplaceAll(command, "<agent>", c.Agent)
}

// GetAgentCommand returns the agent command (e.g., "claude" or "ccode")
func (c *Config) GetAgentCommand() string {
	if c.Agent != "" {
		return c.Agent
	}
	return "claude"
}

// HasTogglePane returns true if a toggle pane is configured
func (c *Config) HasTogglePane() bool {
	return c.TogglePane != nil && c.TogglePane.Name != ""
}

// GetTogglePaneCommand returns the command for the toggle pane with agent expansion
func (c *Config) GetTogglePaneCommand() string {
	if c.TogglePane == nil {
		return ""
	}
	return c.ExpandAgentPlaceholder(c.TogglePane.Command)
}

// AddBookmark adds a worktree name to bookmarks (at the end)
func (c *Config) AddBookmark(name string) bool {
	if c.IsBookmarked(name) {
		return false
	}
	c.Bookmarks = append(c.Bookmarks, name)
	return true
}

// RemoveBookmark removes a worktree name from bookmarks
func (c *Config) RemoveBookmark(name string) bool {
	idx := c.GetBookmarkIndex(name)
	if idx == -1 {
		return false
	}
	c.Bookmarks = append(c.Bookmarks[:idx], c.Bookmarks[idx+1:]...)
	return true
}

// IsBookmarked returns true if the worktree is bookmarked
func (c *Config) IsBookmarked(name string) bool {
	return c.GetBookmarkIndex(name) != -1
}

// GetBookmarkIndex returns the 0-based index of the bookmark, or -1 if not found
func (c *Config) GetBookmarkIndex(name string) int {
	for i, b := range c.Bookmarks {
		if b == name {
			return i
		}
	}
	return -1
}

// GetIndicatorStateDir returns the indicator state directory (with ~ expanded)
func (c *Config) GetIndicatorStateDir() string {
	if c.Indicators == nil || c.Indicators.StateDir == "" {
		return "~/.wt/indicators"
	}
	return c.Indicators.StateDir
}

// HasIndicators returns true if indicators are configured
func (c *Config) HasIndicators() bool {
	return c.Indicators != nil && len(c.Indicators.Definitions) > 0
}

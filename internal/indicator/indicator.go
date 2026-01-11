package indicator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Definition describes how to display a particular indicator type
type Definition struct {
	Name     string            `yaml:"name"`
	Symbols  map[string]string `yaml:"symbols"`  // state -> symbol
	Colors   map[string]string `yaml:"colors"`   // state -> color
	Priority int               `yaml:"priority"` // lower = shown first
}

// State represents the indicator state file for a worktree
type State struct {
	Indicators map[string]string `json:"indicators"` // indicator name -> state
	UpdatedAt  time.Time         `json:"updated_at"`
}

// Result is a resolved indicator ready for display
type Result struct {
	Name   string
	State  string
	Symbol string
	Color  string
}

// Manager handles loading and writing indicator state
type Manager struct {
	stateDir    string
	definitions []Definition
}

// NewManager creates a new indicator manager
func NewManager(stateDir string, definitions []Definition) *Manager {
	// Expand ~ in stateDir
	if strings.HasPrefix(stateDir, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			stateDir = filepath.Join(home, stateDir[1:])
		}
	}

	return &Manager{
		stateDir:    stateDir,
		definitions: definitions,
	}
}

// GetStateDir returns the expanded state directory path
func (m *Manager) GetStateDir() string {
	return m.stateDir
}

// stateFilePath returns the path to a worktree's state file
func (m *Manager) stateFilePath(worktree string) string {
	// Sanitize worktree name for filesystem
	safe := strings.ReplaceAll(worktree, "/", "_")
	safe = strings.ReplaceAll(safe, "\\", "_")
	return filepath.Join(m.stateDir, safe+".json")
}

// loadState reads the state file for a worktree
func (m *Manager) loadState(worktree string) (*State, error) {
	path := m.stateFilePath(worktree)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// saveState writes the state file for a worktree
func (m *Manager) saveState(worktree string, state *State) error {
	// Ensure state directory exists
	if err := os.MkdirAll(m.stateDir, 0755); err != nil {
		return err
	}

	state.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.stateFilePath(worktree), data, 0644)
}

// resolveSymbol returns the symbol for a given state
func (d *Definition) resolveSymbol(state string) string {
	if sym, ok := d.Symbols[state]; ok {
		return sym
	}
	if sym, ok := d.Symbols["_default"]; ok {
		return sym
	}
	return "○"
}

// resolveColor returns the color for a given state
func (d *Definition) resolveColor(state string) string {
	if col, ok := d.Colors[state]; ok {
		return col
	}
	if col, ok := d.Colors["_default"]; ok {
		return col
	}
	return "240"
}

// GetIndicators returns all resolved indicators for a worktree
func (m *Manager) GetIndicators(worktree string) []Result {
	state, _ := m.loadState(worktree)

	// Sort definitions by priority
	sorted := make([]Definition, len(m.definitions))
	copy(sorted, m.definitions)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})

	results := make([]Result, 0, len(sorted))
	for _, def := range sorted {
		stateVal := ""
		if state != nil && state.Indicators != nil {
			stateVal = state.Indicators[def.Name]
		}

		results = append(results, Result{
			Name:   def.Name,
			State:  stateVal,
			Symbol: def.resolveSymbol(stateVal),
			Color:  def.resolveColor(stateVal),
		})
	}

	return results
}

// SetIndicator sets the state for a specific indicator on a worktree
func (m *Manager) SetIndicator(worktree, indicator, stateVal string) error {
	state, err := m.loadState(worktree)
	if err != nil || state == nil {
		state = &State{
			Indicators: make(map[string]string),
		}
	}

	if state.Indicators == nil {
		state.Indicators = make(map[string]string)
	}

	state.Indicators[indicator] = stateVal
	return m.saveState(worktree, state)
}

// ClearIndicators clears indicator(s) for a worktree
// If indicator is empty, clears all indicators
func (m *Manager) ClearIndicators(worktree, indicator string) error {
	if indicator == "" {
		// Clear all - remove the state file
		path := m.stateFilePath(worktree)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}

	// Clear specific indicator
	state, err := m.loadState(worktree)
	if err != nil {
		// File doesn't exist, nothing to clear
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if state.Indicators != nil {
		delete(state.Indicators, indicator)
	}

	return m.saveState(worktree, state)
}

// GetAllStates returns the raw indicator states for a worktree
func (m *Manager) GetAllStates(worktree string) (map[string]string, error) {
	state, err := m.loadState(worktree)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, err
	}

	if state.Indicators == nil {
		return make(map[string]string), nil
	}

	return state.Indicators, nil
}

// ListWorktreesWithState returns all worktrees that have state files
func (m *Manager) ListWorktreesWithState() ([]string, error) {
	entries, err := os.ReadDir(m.stateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var worktrees []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".json") {
			worktrees = append(worktrees, strings.TrimSuffix(name, ".json"))
		}
	}

	return worktrees, nil
}

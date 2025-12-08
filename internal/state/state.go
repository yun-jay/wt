package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const (
	StateFile  = ".wt-state.json"
	MaxHistory = 20
)

// State holds runtime state like recently visited worktrees
type State struct {
	History     []string  `json:"history"`
	LastUpdated time.Time `json:"last_updated"`
}

// LoadState loads state from the project root, creating empty state if not found
func LoadState(projectRoot string) (*State, error) {
	statePath := filepath.Join(projectRoot, StateFile)

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{
				History:     []string{},
				LastUpdated: time.Now(),
			}, nil
		}
		return nil, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// Save writes the state to disk
func (s *State) Save(projectRoot string) error {
	s.LastUpdated = time.Now()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	statePath := filepath.Join(projectRoot, StateFile)
	return os.WriteFile(statePath, data, 0644)
}

// RecordVisit adds a worktree to the front of history, deduping and limiting size
func (s *State) RecordVisit(worktree string) {
	// Remove existing entry if present
	newHistory := []string{worktree}
	for _, wt := range s.History {
		if wt != worktree {
			newHistory = append(newHistory, wt)
		}
	}

	// Limit to MaxHistory entries
	if len(newHistory) > MaxHistory {
		newHistory = newHistory[:MaxHistory]
	}

	s.History = newHistory
}

// GetSortedWorktrees returns worktrees sorted by recency (most recent first)
// Worktrees not in history are appended at the end in their original order
func (s *State) GetSortedWorktrees(all []string) []string {
	// Create a position map for history
	position := make(map[string]int)
	for i, wt := range s.History {
		position[wt] = i
	}

	// Separate into known (in history) and unknown
	var known, unknown []string
	for _, wt := range all {
		if _, ok := position[wt]; ok {
			known = append(known, wt)
		} else {
			unknown = append(unknown, wt)
		}
	}

	// Sort known by their position in history
	sortByPosition(known, position)

	// Combine: known first, then unknown
	return append(known, unknown...)
}

// sortByPosition sorts strings by their position in the position map (insertion sort)
func sortByPosition(items []string, position map[string]int) {
	for i := 1; i < len(items); i++ {
		j := i
		for j > 0 && position[items[j]] < position[items[j-1]] {
			items[j], items[j-1] = items[j-1], items[j]
			j--
		}
	}
}

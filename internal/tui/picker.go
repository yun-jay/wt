package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"
)

var (
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Bold(true)
	normalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	titleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true)
)

// Indicator represents a status indicator to display
type Indicator struct {
	Symbol string
	Color  string
}

// Item represents an item in the picker
type Item struct {
	Name         string
	Description  string
	Value        interface{}
	Indicators   []Indicator // Status indicators to display before the name
	IndicatorKey string      // Key for looking up live indicator updates (e.g., worktree name)
}

// PickerModel is the model for the interactive picker
type PickerModel struct {
	items     []Item
	filtered  []Item
	cursor    int
	selected  *Item
	textInput textinput.Model
	title     string
	cancelled bool
}

// NewPicker creates a new picker with the given items
func NewPicker(title string, items []Item) PickerModel {
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.Focus()

	return PickerModel{
		items:     items,
		filtered:  items,
		cursor:    0,
		textInput: ti,
		title:     title,
	}
}

func (m PickerModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m PickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				m.selected = &m.filtered[m.cursor]
			}
			return m, tea.Quit
		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(m.filtered) - 1 // Wrap to last
			}
			return m, nil
		case "down", "ctrl+n":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			} else {
				m.cursor = 0 // Wrap to first
			}
			return m, nil
		}
	}

	// Update text input
	m.textInput, cmd = m.textInput.Update(msg)

	// Filter items
	query := strings.ToLower(m.textInput.Value())
	m.filtered = []Item{}
	for _, item := range m.items {
		if query == "" || strings.Contains(strings.ToLower(item.Name), query) ||
			strings.Contains(strings.ToLower(item.Description), query) {
			m.filtered = append(m.filtered, item)
		}
	}

	// Reset cursor if out of bounds
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}

	return m, cmd
}

func (m PickerModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(m.title) + "\n\n")
	b.WriteString(m.textInput.View() + "\n\n")

	for i, item := range m.filtered {
		cursor := "  "
		style := normalStyle
		if i == m.cursor {
			cursor = "> "
			style = selectedStyle
		}

		line := cursor
		// Render indicators before the name
		for _, ind := range item.Indicators {
			indStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ind.Color))
			line += indStyle.Render(ind.Symbol)
		}
		if len(item.Indicators) > 0 {
			line += " "
		}
		line += style.Render(item.Name)
		if item.Description != "" {
			line += " " + dimStyle.Render(item.Description)
		}
		b.WriteString(line + "\n")
	}

	if len(m.filtered) == 0 {
		b.WriteString(dimStyle.Render("  No matches found") + "\n")
	}

	b.WriteString("\n" + dimStyle.Render("↑/↓/ctrl+n/p navigate • enter select • esc cancel"))

	return b.String()
}

// Selected returns the selected item or nil if cancelled
func (m PickerModel) Selected() *Item {
	return m.selected
}

// Cancelled returns true if the user cancelled
func (m PickerModel) Cancelled() bool {
	return m.cancelled
}

// RunPicker runs the picker and returns the selected item
func RunPicker(title string, items []Item) (*Item, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("no items to pick from")
	}

	p := tea.NewProgram(NewPicker(title, items))
	m, err := p.Run()
	if err != nil {
		return nil, err
	}

	model := m.(PickerModel)
	if model.Cancelled() {
		return nil, nil
	}

	return model.Selected(), nil
}

// DeleteHandler is a function called when an item is deleted
type DeleteHandler func(item Item) error

// PickerWithDeleteModel is a picker that supports deleting items
type PickerWithDeleteModel struct {
	items         []Item
	filtered      []Item
	cursor        int
	selected      *Item
	textInput     textinput.Model
	title         string
	cancelled     bool
	deleteHandler DeleteHandler
	deleted       bool
	emptyMessage  string
}

// NewPickerWithDelete creates a picker with delete support
func NewPickerWithDelete(title string, items []Item, deleteHandler DeleteHandler) PickerWithDeleteModel {
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.Focus()

	return PickerWithDeleteModel{
		items:         items,
		filtered:      items,
		cursor:        0,
		textInput:     ti,
		title:         title,
		deleteHandler: deleteHandler,
	}
}

func (m PickerWithDeleteModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m PickerWithDeleteModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				m.selected = &m.filtered[m.cursor]
			}
			return m, tea.Quit
		case "x", "d":
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) && m.deleteHandler != nil {
				item := m.filtered[m.cursor]
				if err := m.deleteHandler(item); err == nil {
					// Remove from items list
					m.items = removeItem(m.items, item)
					m.filtered = removeItem(m.filtered, item)
					m.deleted = true
					// Adjust cursor if needed
					if m.cursor >= len(m.filtered) {
						m.cursor = max(0, len(m.filtered)-1)
					}
				}
			}
			return m, nil
		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(m.filtered) - 1
			}
			return m, nil
		case "down", "ctrl+n":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}
			return m, nil
		}
	}

	// Update text input
	m.textInput, cmd = m.textInput.Update(msg)

	// Filter items
	query := strings.ToLower(m.textInput.Value())
	m.filtered = []Item{}
	for _, item := range m.items {
		if query == "" || strings.Contains(strings.ToLower(item.Name), query) ||
			strings.Contains(strings.ToLower(item.Description), query) {
			m.filtered = append(m.filtered, item)
		}
	}

	// Reset cursor if out of bounds
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}

	return m, cmd
}

func (m PickerWithDeleteModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(m.title) + "\n\n")

	// Show empty message if no items at all
	if len(m.items) == 0 {
		if m.emptyMessage != "" {
			b.WriteString(dimStyle.Render("  "+m.emptyMessage) + "\n")
		} else {
			b.WriteString(dimStyle.Render("  No items") + "\n")
		}
		b.WriteString("\n" + dimStyle.Render("esc: cancel"))
		return b.String()
	}

	b.WriteString(m.textInput.View() + "\n\n")

	for i, item := range m.filtered {
		cursor := "  "
		style := normalStyle
		if i == m.cursor {
			cursor = "> "
			style = selectedStyle
		}

		line := cursor
		// Render indicators before the name
		for _, ind := range item.Indicators {
			indStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ind.Color))
			line += indStyle.Render(ind.Symbol)
		}
		if len(item.Indicators) > 0 {
			line += " "
		}
		line += style.Render(item.Name)
		if item.Description != "" {
			line += " " + dimStyle.Render(item.Description)
		}
		b.WriteString(line + "\n")
	}

	if len(m.filtered) == 0 {
		b.WriteString(dimStyle.Render("  No matches found") + "\n")
	}

	b.WriteString("\n" + dimStyle.Render("enter: switch  x: remove  esc: cancel"))

	return b.String()
}

// Selected returns the selected item or nil if cancelled
func (m PickerWithDeleteModel) Selected() *Item {
	return m.selected
}

// Cancelled returns true if the user cancelled
func (m PickerWithDeleteModel) Cancelled() bool {
	return m.cancelled
}

// RunPickerWithDelete runs the picker with delete support
func RunPickerWithDelete(title string, items []Item, deleteHandler DeleteHandler) (*Item, error) {
	return RunPickerWithDeleteAndEmpty(title, items, deleteHandler, "")
}

// RunPickerWithDeleteAndEmpty runs the picker with delete support and empty state handling
func RunPickerWithDeleteAndEmpty(title string, items []Item, deleteHandler DeleteHandler, emptyMessage string) (*Item, error) {
	model := NewPickerWithDelete(title, items, deleteHandler)
	model.emptyMessage = emptyMessage

	p := tea.NewProgram(model)
	m, err := p.Run()
	if err != nil {
		return nil, err
	}

	result := m.(PickerWithDeleteModel)
	if result.Cancelled() {
		return nil, nil
	}

	return result.Selected(), nil
}

func removeItem(items []Item, toRemove Item) []Item {
	var result []Item
	for _, item := range items {
		if item.Name != toRemove.Name {
			result = append(result, item)
		}
	}
	return result
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// IndicatorUpdateMsg is sent when indicator files change
type IndicatorUpdateMsg struct {
	Worktree   string
	Indicators []Indicator
}

// IndicatorRefreshFunc is called to refresh indicators for a worktree
type IndicatorRefreshFunc func(worktree string) []Indicator

// PickerWithIndicatorsModel extends PickerModel with live indicator updates
type PickerWithIndicatorsModel struct {
	items       []Item
	filtered    []Item
	cursor      int
	selected    *Item
	textInput   textinput.Model
	title       string
	cancelled   bool
	watcher     *fsnotify.Watcher
	stateDir    string
	refreshFunc IndicatorRefreshFunc
}

// NewPickerWithIndicators creates a picker that watches for indicator updates
func NewPickerWithIndicators(title string, items []Item, stateDir string, refreshFunc IndicatorRefreshFunc) PickerWithIndicatorsModel {
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.Focus()

	return PickerWithIndicatorsModel{
		items:       items,
		filtered:    items,
		cursor:      0,
		textInput:   ti,
		title:       title,
		stateDir:    stateDir,
		refreshFunc: refreshFunc,
	}
}

func (m PickerWithIndicatorsModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.watchIndicators())
}

func (m PickerWithIndicatorsModel) watchIndicators() tea.Cmd {
	return func() tea.Msg {
		if m.stateDir == "" || m.refreshFunc == nil {
			return nil
		}

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return nil
		}

		if err := watcher.Add(m.stateDir); err != nil {
			watcher.Close()
			return nil
		}

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return nil
				}
				if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
					// Extract worktree name from filename
					filename := filepath.Base(event.Name)
					if strings.HasSuffix(filename, ".json") {
						worktree := strings.TrimSuffix(filename, ".json")
						return IndicatorUpdateMsg{
							Worktree:   worktree,
							Indicators: m.refreshFunc(worktree),
						}
					}
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return nil
				}
			}
		}
	}
}

func (m PickerWithIndicatorsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case IndicatorUpdateMsg:
		// Update indicators for the matching item
		for i := range m.items {
			if m.items[i].IndicatorKey == msg.Worktree {
				m.items[i].Indicators = msg.Indicators
			}
		}
		// Update filtered list too
		for i := range m.filtered {
			if m.filtered[i].IndicatorKey == msg.Worktree {
				m.filtered[i].Indicators = msg.Indicators
			}
		}
		// Continue watching
		return m, m.watchIndicators()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			if m.watcher != nil {
				m.watcher.Close()
			}
			return m, tea.Quit
		case "enter":
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				m.selected = &m.filtered[m.cursor]
			}
			if m.watcher != nil {
				m.watcher.Close()
			}
			return m, tea.Quit
		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(m.filtered) - 1
			}
			return m, nil
		case "down", "ctrl+n":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}
			return m, nil
		}
	}

	// Update text input
	m.textInput, cmd = m.textInput.Update(msg)

	// Filter items
	query := strings.ToLower(m.textInput.Value())
	m.filtered = []Item{}
	for _, item := range m.items {
		if query == "" || strings.Contains(strings.ToLower(item.Name), query) ||
			strings.Contains(strings.ToLower(item.Description), query) {
			m.filtered = append(m.filtered, item)
		}
	}

	// Reset cursor if out of bounds
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}

	return m, cmd
}

func (m PickerWithIndicatorsModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(m.title) + "\n\n")
	b.WriteString(m.textInput.View() + "\n\n")

	for i, item := range m.filtered {
		cursor := "  "
		style := normalStyle
		if i == m.cursor {
			cursor = "> "
			style = selectedStyle
		}

		line := cursor
		// Render indicators before the name
		for _, ind := range item.Indicators {
			indStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ind.Color))
			line += indStyle.Render(ind.Symbol)
		}
		if len(item.Indicators) > 0 {
			line += " "
		}
		line += style.Render(item.Name)
		if item.Description != "" {
			line += " " + dimStyle.Render(item.Description)
		}
		b.WriteString(line + "\n")
	}

	if len(m.filtered) == 0 {
		b.WriteString(dimStyle.Render("  No matches found") + "\n")
	}

	b.WriteString("\n" + dimStyle.Render("↑/↓/ctrl+n/p navigate • enter select • esc cancel"))

	return b.String()
}

// Selected returns the selected item or nil if cancelled
func (m PickerWithIndicatorsModel) Selected() *Item {
	return m.selected
}

// Cancelled returns true if the user cancelled
func (m PickerWithIndicatorsModel) Cancelled() bool {
	return m.cancelled
}

// RunPickerWithIndicators runs the picker with live indicator updates
func RunPickerWithIndicators(title string, items []Item, stateDir string, refreshFunc IndicatorRefreshFunc) (*Item, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("no items to pick from")
	}

	p := tea.NewProgram(NewPickerWithIndicators(title, items, stateDir, refreshFunc))
	m, err := p.Run()
	if err != nil {
		return nil, err
	}

	model := m.(PickerWithIndicatorsModel)
	if model.Cancelled() {
		return nil, nil
	}

	return model.Selected(), nil
}

// MultiSelectPickerModel is a picker that allows selecting multiple items
type MultiSelectPickerModel struct {
	items     []Item
	filtered  []Item
	cursor    int
	selected  map[int]bool // Maps original item index to selection state
	textInput textinput.Model
	title     string
	cancelled bool
}

// NewMultiSelectPicker creates a new multi-select picker with the given items
func NewMultiSelectPicker(title string, items []Item) MultiSelectPickerModel {
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.Focus()

	return MultiSelectPickerModel{
		items:     items,
		filtered:  items,
		cursor:    0,
		selected:  make(map[int]bool),
		textInput: ti,
		title:     title,
	}
}

func (m MultiSelectPickerModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m MultiSelectPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			return m, tea.Quit
		case " ": // Space bar toggles selection
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				origIdx := m.getOriginalIndex(m.filtered[m.cursor])
				if origIdx >= 0 {
					m.selected[origIdx] = !m.selected[origIdx]
				}
			}
			return m, nil
		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(m.filtered) - 1 // Wrap to last
			}
			return m, nil
		case "down", "ctrl+n":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			} else {
				m.cursor = 0 // Wrap to first
			}
			return m, nil
		}
	}

	// Update text input
	m.textInput, cmd = m.textInput.Update(msg)

	// Filter items
	query := strings.ToLower(m.textInput.Value())
	m.filtered = []Item{}
	for _, item := range m.items {
		if query == "" || strings.Contains(strings.ToLower(item.Name), query) ||
			strings.Contains(strings.ToLower(item.Description), query) {
			m.filtered = append(m.filtered, item)
		}
	}

	// Reset cursor if out of bounds
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}

	return m, cmd
}

// getOriginalIndex finds the index of an item in the original items slice
func (m MultiSelectPickerModel) getOriginalIndex(item Item) int {
	for i, orig := range m.items {
		if orig.Name == item.Name {
			return i
		}
	}
	return -1
}

func (m MultiSelectPickerModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(m.title) + "\n\n")
	b.WriteString(m.textInput.View() + "\n\n")

	for i, item := range m.filtered {
		cursor := "  "
		style := normalStyle
		if i == m.cursor {
			cursor = "> "
			style = selectedStyle
		}

		// Checkbox rendering
		origIdx := m.getOriginalIndex(item)
		checkbox := "[ ] "
		if m.selected[origIdx] {
			checkbox = "[x] "
		}

		line := cursor + checkbox

		// Render indicators before the name
		for _, ind := range item.Indicators {
			indStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ind.Color))
			line += indStyle.Render(ind.Symbol)
		}
		if len(item.Indicators) > 0 {
			line += " "
		}

		line += style.Render(item.Name)
		if item.Description != "" {
			line += " " + dimStyle.Render(item.Description)
		}
		b.WriteString(line + "\n")
	}

	if len(m.filtered) == 0 {
		b.WriteString(dimStyle.Render("  No matches found") + "\n")
	}

	// Footer with selection count
	selectedCount := m.getSelectedCount()
	footer := fmt.Sprintf("↑/↓ navigate • space toggle • enter confirm (%d selected) • esc cancel", selectedCount)
	b.WriteString("\n" + dimStyle.Render(footer))

	return b.String()
}

func (m MultiSelectPickerModel) getSelectedCount() int {
	count := 0
	for _, isSelected := range m.selected {
		if isSelected {
			count++
		}
	}
	return count
}

// SelectedItems returns all selected items
func (m MultiSelectPickerModel) SelectedItems() []Item {
	var result []Item
	for idx, isSelected := range m.selected {
		if isSelected && idx < len(m.items) {
			result = append(result, m.items[idx])
		}
	}
	return result
}

// Cancelled returns true if the user cancelled
func (m MultiSelectPickerModel) Cancelled() bool {
	return m.cancelled
}

// RunMultiSelectPicker runs the multi-select picker and returns selected items
func RunMultiSelectPicker(title string, items []Item) ([]Item, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("no items to pick from")
	}

	p := tea.NewProgram(NewMultiSelectPicker(title, items))
	m, err := p.Run()
	if err != nil {
		return nil, err
	}

	model := m.(MultiSelectPickerModel)
	if model.Cancelled() {
		return nil, nil
	}

	return model.SelectedItems(), nil
}

package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewPicker(t *testing.T) {
	items := []Item{
		{Name: "item1", Description: "desc1"},
		{Name: "item2", Description: "desc2"},
	}

	picker := NewPicker("Test Title", items)

	if picker.title != "Test Title" {
		t.Errorf("title = %s, want Test Title", picker.title)
	}

	if len(picker.items) != 2 {
		t.Errorf("items count = %d, want 2", len(picker.items))
	}

	if picker.cursor != 0 {
		t.Errorf("initial cursor = %d, want 0", picker.cursor)
	}
}

func TestPickerNavigationDown(t *testing.T) {
	items := []Item{
		{Name: "item1"},
		{Name: "item2"},
		{Name: "item3"},
	}

	picker := NewPicker("Test", items)

	// Move down
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	// Simulate ctrl+n
	msg = tea.KeyMsg{Type: tea.KeyCtrlN}
	model, _ := picker.Update(msg)
	picker = model.(PickerModel)

	if picker.cursor != 1 {
		t.Errorf("cursor after down = %d, want 1", picker.cursor)
	}

	// Move down again
	model, _ = picker.Update(msg)
	picker = model.(PickerModel)

	if picker.cursor != 2 {
		t.Errorf("cursor after second down = %d, want 2", picker.cursor)
	}

	// Move down at last item should wrap to first
	model, _ = picker.Update(msg)
	picker = model.(PickerModel)

	if picker.cursor != 0 {
		t.Errorf("cursor after wrap = %d, want 0 (wrap around)", picker.cursor)
	}
}

func TestPickerNavigationUp(t *testing.T) {
	items := []Item{
		{Name: "item1"},
		{Name: "item2"},
		{Name: "item3"},
	}

	picker := NewPicker("Test", items)

	// Move up at first item should wrap to last
	msg := tea.KeyMsg{Type: tea.KeyCtrlP}
	model, _ := picker.Update(msg)
	picker = model.(PickerModel)

	if picker.cursor != 2 {
		t.Errorf("cursor after up from first = %d, want 2 (wrap around)", picker.cursor)
	}

	// Move up again
	model, _ = picker.Update(msg)
	picker = model.(PickerModel)

	if picker.cursor != 1 {
		t.Errorf("cursor after up = %d, want 1", picker.cursor)
	}
}

func TestPickerSelection(t *testing.T) {
	items := []Item{
		{Name: "item1", Value: "value1"},
		{Name: "item2", Value: "value2"},
	}

	picker := NewPicker("Test", items)

	// Move to second item
	msg := tea.KeyMsg{Type: tea.KeyCtrlN}
	model, _ := picker.Update(msg)
	picker = model.(PickerModel)

	// Select
	msg = tea.KeyMsg{Type: tea.KeyEnter}
	model, _ = picker.Update(msg)
	picker = model.(PickerModel)

	selected := picker.Selected()
	if selected == nil {
		t.Fatal("Selected() returned nil")
	}

	if selected.Name != "item2" {
		t.Errorf("Selected name = %s, want item2", selected.Name)
	}
}

func TestPickerCancel(t *testing.T) {
	items := []Item{
		{Name: "item1"},
	}

	picker := NewPicker("Test", items)

	// Press escape
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	model, _ := picker.Update(msg)
	picker = model.(PickerModel)

	if !picker.Cancelled() {
		t.Error("Cancelled() = false, want true after Esc")
	}

	if picker.Selected() != nil {
		t.Error("Selected() should be nil after cancel")
	}
}

func TestPickerFiltering(t *testing.T) {
	items := []Item{
		{Name: "apple"},
		{Name: "banana"},
		{Name: "cherry"},
	}

	picker := NewPicker("Test", items)

	// Type 'a' to filter
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	model, _ := picker.Update(msg)
	picker = model.(PickerModel)

	// Should have 2 filtered items (apple, banana)
	if len(picker.filtered) != 2 {
		t.Errorf("filtered count after 'a' = %d, want 2", len(picker.filtered))
	}

	// Type 'p' to narrow down
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	model, _ = picker.Update(msg)
	picker = model.(PickerModel)

	// Should have 1 filtered item (apple)
	if len(picker.filtered) != 1 {
		t.Errorf("filtered count after 'ap' = %d, want 1", len(picker.filtered))
	}

	if picker.filtered[0].Name != "apple" {
		t.Errorf("filtered item = %s, want apple", picker.filtered[0].Name)
	}
}

func TestPickerEmptyFilter(t *testing.T) {
	items := []Item{
		{Name: "item1"},
		{Name: "item2"},
	}

	picker := NewPicker("Test", items)

	// Type something that matches nothing
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}}
	model, _ := picker.Update(msg)
	picker = model.(PickerModel)

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}}
	model, _ = picker.Update(msg)
	picker = model.(PickerModel)

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}}
	model, _ = picker.Update(msg)
	picker = model.(PickerModel)

	if len(picker.filtered) != 0 {
		t.Errorf("filtered count = %d, want 0", len(picker.filtered))
	}
}

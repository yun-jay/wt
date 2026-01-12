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

// MultiSelectPicker tests

func TestMultiSelectPickerToggle(t *testing.T) {
	items := []Item{
		{Name: "item1", Value: "value1"},
		{Name: "item2", Value: "value2"},
		{Name: "item3", Value: "value3"},
	}

	picker := NewMultiSelectPicker("Test", items)

	// Toggle first item with space
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	model, _ := picker.Update(msg)
	picker = model.(MultiSelectPickerModel)

	selected := picker.SelectedItems()
	if len(selected) != 1 {
		t.Errorf("selected count = %d, want 1", len(selected))
	}
	if selected[0].Name != "item1" {
		t.Errorf("selected item = %s, want item1", selected[0].Name)
	}

	// Toggle first item again (deselect)
	model, _ = picker.Update(msg)
	picker = model.(MultiSelectPickerModel)

	selected = picker.SelectedItems()
	if len(selected) != 0 {
		t.Errorf("selected count after deselect = %d, want 0", len(selected))
	}
}

func TestMultiSelectPickerMultipleSelections(t *testing.T) {
	items := []Item{
		{Name: "item1"},
		{Name: "item2"},
		{Name: "item3"},
	}

	picker := NewMultiSelectPicker("Test", items)

	// Select item1
	spaceMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	model, _ := picker.Update(spaceMsg)
	picker = model.(MultiSelectPickerModel)

	// Move down and select item2
	downMsg := tea.KeyMsg{Type: tea.KeyCtrlN}
	model, _ = picker.Update(downMsg)
	picker = model.(MultiSelectPickerModel)

	model, _ = picker.Update(spaceMsg)
	picker = model.(MultiSelectPickerModel)

	// Move down and select item3
	model, _ = picker.Update(downMsg)
	picker = model.(MultiSelectPickerModel)

	model, _ = picker.Update(spaceMsg)
	picker = model.(MultiSelectPickerModel)

	selected := picker.SelectedItems()
	if len(selected) != 3 {
		t.Errorf("selected count = %d, want 3", len(selected))
	}
}

func TestMultiSelectPickerSelectionPersistsAcrossFilter(t *testing.T) {
	items := []Item{
		{Name: "apple"},
		{Name: "banana"},
		{Name: "cherry"},
	}

	picker := NewMultiSelectPicker("Test", items)

	// Select apple
	spaceMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	model, _ := picker.Update(spaceMsg)
	picker = model.(MultiSelectPickerModel)

	// Filter to show only 'banana'
	bMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}
	model, _ = picker.Update(bMsg)
	picker = model.(MultiSelectPickerModel)

	// Clear filter by pressing backspace
	backspaceMsg := tea.KeyMsg{Type: tea.KeyBackspace}
	model, _ = picker.Update(backspaceMsg)
	picker = model.(MultiSelectPickerModel)

	// apple should still be selected
	selected := picker.SelectedItems()
	if len(selected) != 1 {
		t.Errorf("selection not preserved: got %d items, want 1", len(selected))
	}
	if len(selected) > 0 && selected[0].Name != "apple" {
		t.Errorf("wrong item selected: got %s, want apple", selected[0].Name)
	}
}

func TestMultiSelectPickerCancel(t *testing.T) {
	items := []Item{{Name: "item1"}}
	picker := NewMultiSelectPicker("Test", items)

	// Select item
	spaceMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	model, _ := picker.Update(spaceMsg)
	picker = model.(MultiSelectPickerModel)

	// Cancel
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	model, _ = picker.Update(escMsg)
	picker = model.(MultiSelectPickerModel)

	if !picker.Cancelled() {
		t.Error("Cancelled() = false, want true")
	}
}

func TestMultiSelectPickerConfirmEmpty(t *testing.T) {
	items := []Item{{Name: "item1"}}
	picker := NewMultiSelectPicker("Test", items)

	// Confirm without selecting anything
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	model, _ := picker.Update(enterMsg)
	picker = model.(MultiSelectPickerModel)

	selected := picker.SelectedItems()
	if len(selected) != 0 {
		t.Errorf("selected count = %d, want 0", len(selected))
	}
	if picker.Cancelled() {
		t.Error("should not be cancelled when pressing enter")
	}
}

func TestMultiSelectPickerNavigation(t *testing.T) {
	items := []Item{
		{Name: "item1"},
		{Name: "item2"},
		{Name: "item3"},
	}

	picker := NewMultiSelectPicker("Test", items)

	// Move down
	downMsg := tea.KeyMsg{Type: tea.KeyCtrlN}
	model, _ := picker.Update(downMsg)
	picker = model.(MultiSelectPickerModel)

	if picker.cursor != 1 {
		t.Errorf("cursor after down = %d, want 1", picker.cursor)
	}

	// Move down to last
	model, _ = picker.Update(downMsg)
	picker = model.(MultiSelectPickerModel)

	if picker.cursor != 2 {
		t.Errorf("cursor after second down = %d, want 2", picker.cursor)
	}

	// Move down at last should wrap to first
	model, _ = picker.Update(downMsg)
	picker = model.(MultiSelectPickerModel)

	if picker.cursor != 0 {
		t.Errorf("cursor after wrap = %d, want 0", picker.cursor)
	}

	// Move up at first should wrap to last
	upMsg := tea.KeyMsg{Type: tea.KeyCtrlP}
	model, _ = picker.Update(upMsg)
	picker = model.(MultiSelectPickerModel)

	if picker.cursor != 2 {
		t.Errorf("cursor after up wrap = %d, want 2", picker.cursor)
	}
}

func TestMultiSelectPickerGetSelectedCount(t *testing.T) {
	items := []Item{
		{Name: "item1"},
		{Name: "item2"},
		{Name: "item3"},
	}

	picker := NewMultiSelectPicker("Test", items)

	if picker.getSelectedCount() != 0 {
		t.Errorf("initial selected count = %d, want 0", picker.getSelectedCount())
	}

	// Select two items
	spaceMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	model, _ := picker.Update(spaceMsg)
	picker = model.(MultiSelectPickerModel)

	downMsg := tea.KeyMsg{Type: tea.KeyCtrlN}
	model, _ = picker.Update(downMsg)
	picker = model.(MultiSelectPickerModel)

	model, _ = picker.Update(spaceMsg)
	picker = model.(MultiSelectPickerModel)

	if picker.getSelectedCount() != 2 {
		t.Errorf("selected count after selections = %d, want 2", picker.getSelectedCount())
	}
}

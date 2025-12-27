package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestUnifiedFormModel_AddTextInput(t *testing.T) {
	form := NewUnifiedFormModel("Test Form")
	form.AddTextInput("name", "Name", "Enter name", true)

	if len(form.Components) != 1 {
		t.Errorf("AddTextInput() len(Components) = %d, want 1", len(form.Components))
	}

	comp := form.Components[0]
	if comp.Type != ComponentTextInput {
		t.Errorf("AddTextInput() Type = %v, want ComponentTextInput", comp.Type)
	}
	if comp.ID != "name" {
		t.Errorf("AddTextInput() ID = %q, want %q", comp.ID, "name")
	}
	if comp.Label != "Name" {
		t.Errorf("AddTextInput() Label = %q, want %q", comp.Label, "Name")
	}
	if comp.TextInput == nil {
		t.Error("AddTextInput() TextInput is nil")
	}
	if !comp.Required {
		t.Error("AddTextInput() Required = false, want true")
	}
}

func TestUnifiedFormModel_AddCheckboxList(t *testing.T) {
	form := NewUnifiedFormModel("Test Form")
	items := []CheckboxItem{
		{Label: "Item 1", Checked: false, Enabled: true},
		{Label: "Item 2", Checked: true, Enabled: true},
	}
	form.AddCheckboxList("scopes", items)

	if len(form.Components) != 1 {
		t.Errorf("AddCheckboxList() len(Components) = %d, want 1", len(form.Components))
	}

	comp := form.Components[0]
	if comp.Type != ComponentCheckboxList {
		t.Errorf("AddCheckboxList() Type = %v, want ComponentCheckboxList", comp.Type)
	}
	if comp.ID != "scopes" {
		t.Errorf("AddCheckboxList() ID = %q, want %q", comp.ID, "scopes")
	}
	if comp.CheckboxList == nil {
		t.Error("AddCheckboxList() CheckboxList is nil")
	}
}

func TestUnifiedFormModel_AddButtonGroup(t *testing.T) {
	form := NewUnifiedFormModel("Test Form")
	form.AddButtonGroup("actions", []string{"OK", "Cancel"}, 0)

	if len(form.Components) != 1 {
		t.Errorf("AddButtonGroup() len(Components) = %d, want 1", len(form.Components))
	}

	comp := form.Components[0]
	if comp.Type != ComponentButtonGroup {
		t.Errorf("AddButtonGroup() Type = %v, want ComponentButtonGroup", comp.Type)
	}
	if comp.ID != "actions" {
		t.Errorf("AddButtonGroup() ID = %q, want %q", comp.ID, "actions")
	}
	if comp.ButtonGroup == nil {
		t.Error("AddButtonGroup() ButtonGroup is nil")
	}
}

func TestUnifiedFormModel_Navigation(t *testing.T) {
	form := NewUnifiedFormModel("Test Form")
	form.AddTextInput("field1", "Field 1", "Enter value", false)
	form.AddTextInput("field2", "Field 2", "Enter value", false)
	form.AddButtonGroup("actions", []string{"OK"}, 0)

	// Test forward navigation
	if form.FocusedIdx != 0 {
		t.Errorf("Initial FocusedIdx = %d, want 0", form.FocusedIdx)
	}

	form.navigateForward()
	if form.FocusedIdx != 1 {
		t.Errorf("navigateForward() FocusedIdx = %d, want 1", form.FocusedIdx)
	}

	form.navigateForward()
	if form.FocusedIdx != 2 {
		t.Errorf("navigateForward() FocusedIdx = %d, want 2", form.FocusedIdx)
	}

	// Test wrap around
	form.navigateForward()
	if form.FocusedIdx != 0 {
		t.Errorf("navigateForward() with wrap FocusedIdx = %d, want 0", form.FocusedIdx)
	}

	// Test backward navigation
	form.navigateBackward()
	if form.FocusedIdx != 2 {
		t.Errorf("navigateBackward() FocusedIdx = %d, want 2", form.FocusedIdx)
	}
}

func TestUnifiedFormModel_GetValues(t *testing.T) {
	form := NewUnifiedFormModel("Test Form")
	form.AddTextInput("name", "Name", "Enter name", false)
	form.AddCheckboxList("scopes", []CheckboxItem{
		{Label: "Item 1", Checked: true, Enabled: true},
		{Label: "Item 2", Checked: false, Enabled: true},
		{Label: "Item 3", Checked: true, Enabled: true},
	})
	form.AddButtonGroup("actions", []string{"OK", "Cancel"}, 1)

	// Set text input value
	if form.Components[0].TextInput != nil {
		form.Components[0].TextInput.SetValue("Test Name")
	}

	values := form.GetValues()

	// Check text input value
	if name, ok := values["name"].(string); !ok || name != "Test Name" {
		t.Errorf("GetValues() name = %v, want %q", values["name"], "Test Name")
	}

	// Check checkbox list selected indices
	if selected, ok := values["scopes"].([]int); !ok {
		t.Errorf("GetValues() scopes type = %T, want []int", values["scopes"])
	} else {
		if len(selected) != 2 {
			t.Errorf("GetValues() scopes len = %d, want 2", len(selected))
		}
		if selected[0] != 0 || selected[1] != 2 {
			t.Errorf("GetValues() scopes = %v, want [0, 2]", selected)
		}
	}

	// Check button group selected
	if selected, ok := values["actions"].(int); !ok || selected != 1 {
		t.Errorf("GetValues() actions = %v, want 1", values["actions"])
	}
}

func TestUnifiedFormModel_ValidateForm(t *testing.T) {
	form := NewUnifiedFormModel("Test Form")
	form.AddTextInput("name", "Name", "Enter name", true)
	form.AddTextInput("optional", "Optional", "Enter value", false)

	// Test required field validation
	if form.validateForm() {
		t.Error("validateForm() should fail when required field is empty")
	}
	if form.Error == "" {
		t.Error("validateForm() should set Error when validation fails")
	}

	// Set required field value
	if form.Components[0].TextInput != nil {
		form.Components[0].TextInput.SetValue("Test Name")
	}

	if !form.validateForm() {
		t.Error("validateForm() should pass when required field is filled")
	}
	if form.Error != "" {
		t.Errorf("validateForm() Error = %q, want empty", form.Error)
	}
}

func TestUnifiedFormModel_Update_TabNavigation(t *testing.T) {
	form := NewUnifiedFormModel("Test Form")
	form.AddTextInput("field1", "Field 1", "Enter value", false)
	form.AddTextInput("field2", "Field 2", "Enter value", false)

	// Test Tab navigation
	msg := tea.KeyMsg{Type: tea.KeyTab}
	_, cmd := form.Update(msg)

	if form.FocusedIdx != 1 {
		t.Errorf("Update(Tab) FocusedIdx = %d, want 1", form.FocusedIdx)
	}

	// Test Shift+Tab navigation
	msg = tea.KeyMsg{Type: tea.KeyShiftTab}
	_, cmd = form.Update(msg)

	if form.FocusedIdx != 0 {
		t.Errorf("Update(Shift+Tab) FocusedIdx = %d, want 0", form.FocusedIdx)
	}

	_ = cmd // Suppress unused variable warning
}

func TestUnifiedFormModel_Update_Escape(t *testing.T) {
	form := NewUnifiedFormModel("Test Form")
	form.AddTextInput("field1", "Field 1", "Enter value", false)

	cancelled := false
	form.OnCancel = func() {
		cancelled = true
	}

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd := form.Update(msg)

	if !cancelled {
		t.Error("Update(Esc) should call OnCancel")
	}

	_ = cmd // Suppress unused variable warning
}

func TestUnifiedFormModel_GetTextInputValue(t *testing.T) {
	form := NewUnifiedFormModel("Test Form")
	form.AddTextInput("name", "Name", "Enter name", false)

	if form.Components[0].TextInput != nil {
		form.Components[0].TextInput.SetValue("Test Value")
	}

	value := form.GetTextInputValue("name")
	if value != "Test Value" {
		t.Errorf("GetTextInputValue() = %q, want %q", value, "Test Value")
	}

	// Test non-existent ID
	value = form.GetTextInputValue("nonexistent")
	if value != "" {
		t.Errorf("GetTextInputValue(nonexistent) = %q, want empty", value)
	}
}

func TestUnifiedFormModel_GetCheckboxListSelected(t *testing.T) {
	form := NewUnifiedFormModel("Test Form")
	form.AddCheckboxList("scopes", []CheckboxItem{
		{Label: "Item 1", Checked: true, Enabled: true},
		{Label: "Item 2", Checked: false, Enabled: true},
		{Label: "Item 3", Checked: true, Enabled: true},
	})

	selected := form.GetCheckboxListSelected("scopes")
	if len(selected) != 2 {
		t.Errorf("GetCheckboxListSelected() len = %d, want 2", len(selected))
	}
	if selected[0] != 0 || selected[1] != 2 {
		t.Errorf("GetCheckboxListSelected() = %v, want [0, 2]", selected)
	}

	// Test non-existent ID
	selected = form.GetCheckboxListSelected("nonexistent")
	if len(selected) != 0 {
		t.Errorf("GetCheckboxListSelected(nonexistent) = %v, want []", selected)
	}
}

func TestUnifiedFormModel_SetValue(t *testing.T) {
	form := NewUnifiedFormModel("Test Form")
	form.AddTextInput("name", "Name", "Enter name", false)

	form.SetValue("name", "New Value")

	value := form.GetTextInputValue("name")
	if value != "New Value" {
		t.Errorf("SetValue() value = %q, want %q", value, "New Value")
	}
}

func TestUnifiedFormModel_FocusManagement(t *testing.T) {
	form := NewUnifiedFormModel("Test Form")
	form.AddTextInput("field1", "Field 1", "Enter value", false)
	form.AddCheckboxList("scopes", []CheckboxItem{
		{Label: "Item 1", Checked: false, Enabled: true},
	})

	// Test focus component
	form.focusComponent(0)
	if form.Components[0].TextInput == nil || !form.Components[0].TextInput.Focused() {
		t.Error("focusComponent(0) should focus text input")
	}

	form.focusComponent(1)
	if form.Components[1].CheckboxList == nil || !form.Components[1].CheckboxList.HasFocus {
		t.Error("focusComponent(1) should focus checkbox list")
	}

	// Test blur component
	form.blurComponent(0)
	if form.Components[0].TextInput != nil && form.Components[0].TextInput.Focused() {
		t.Error("blurComponent(0) should blur text input")
	}

	form.blurComponent(1)
	if form.Components[1].CheckboxList != nil && form.Components[1].CheckboxList.HasFocus {
		t.Error("blurComponent(1) should blur checkbox list")
	}
}

func TestNewTextInputComponent(t *testing.T) {
	comp := NewTextInputComponent("test", "Test Label", "Enter value", true)

	if comp.Type != ComponentTextInput {
		t.Errorf("NewTextInputComponent() Type = %v, want ComponentTextInput", comp.Type)
	}
	if comp.ID != "test" {
		t.Errorf("NewTextInputComponent() ID = %q, want %q", comp.ID, "test")
	}
	if comp.Label != "Test Label" {
		t.Errorf("NewTextInputComponent() Label = %q, want %q", comp.Label, "Test Label")
	}
	if comp.TextInput == nil {
		t.Error("NewTextInputComponent() TextInput is nil")
	}
	if !comp.Required {
		t.Error("NewTextInputComponent() Required = false, want true")
	}
}

func TestNewCheckboxListComponent(t *testing.T) {
	items := []CheckboxItem{
		{Label: "Item 1", Checked: false, Enabled: true},
	}
	comp := NewCheckboxListComponent("test", items)

	if comp.Type != ComponentCheckboxList {
		t.Errorf("NewCheckboxListComponent() Type = %v, want ComponentCheckboxList", comp.Type)
	}
	if comp.ID != "test" {
		t.Errorf("NewCheckboxListComponent() ID = %q, want %q", comp.ID, "test")
	}
	if comp.CheckboxList == nil {
		t.Error("NewCheckboxListComponent() CheckboxList is nil")
	}
}

func TestNewButtonGroupComponent(t *testing.T) {
	comp := NewButtonGroupComponent("test", []string{"OK", "Cancel"}, 0)

	if comp.Type != ComponentButtonGroup {
		t.Errorf("NewButtonGroupComponent() Type = %v, want ComponentButtonGroup", comp.Type)
	}
	if comp.ID != "test" {
		t.Errorf("NewButtonGroupComponent() ID = %q, want %q", comp.ID, "test")
	}
	if comp.ButtonGroup == nil {
		t.Error("NewButtonGroupComponent() ButtonGroup is nil")
	}
}

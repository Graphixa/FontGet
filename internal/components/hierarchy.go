package components

import (
	"fmt"
	"strings"

	"fontget/internal/ui"

	"github.com/charmbracelet/lipgloss"
)

// HierarchyItem represents an item in a hierarchical list
type HierarchyItem struct {
	Name     string
	Children []HierarchyItem
	Level    int
	Expanded bool
	Style    lipgloss.Style
}

// HierarchyModel represents a hierarchical list model
type HierarchyModel struct {
	Title       string
	Items       []HierarchyItem
	Width       int
	Height      int
	ShowDetails bool
	IndentSize  int
}

// NewHierarchyItem creates a new hierarchy item
func NewHierarchyItem(name string) HierarchyItem {
	return HierarchyItem{
		Name:     name,
		Children: []HierarchyItem{},
		Level:    0,
		Expanded: true,
		Style:    getDefaultItemStyle(),
	}
}

// NewHierarchyModel creates a new hierarchy model
func NewHierarchyModel(title string, items []HierarchyItem) *HierarchyModel {
	return &HierarchyModel{
		Title:       title,
		Items:       items,
		Width:       80,
		Height:      24,
		ShowDetails: false,
		IndentSize:  2,
	}
}

// AddChild adds a child item to a hierarchy item
func (h *HierarchyItem) AddChild(child HierarchyItem) {
	child.Level = h.Level + 1
	child.Style = getChildItemStyle(child.Level)
	h.Children = append(h.Children, child)
}

// AddChildren adds multiple child items
func (h *HierarchyItem) AddChildren(children []HierarchyItem) {
	for _, child := range children {
		h.AddChild(child)
	}
}

// Render renders a hierarchy item
func (h HierarchyItem) Render(indentSize int) string {
	var result strings.Builder

	// For font families (level 0), no indentation
	// For variants (level 1+), use proper indentation with arrows
	if h.Level == 0 {
		// Font family - no indentation, no arrow
		itemText := h.Name
		result.WriteString(h.Style.Render(itemText))
		result.WriteString("\n")
	} else {
		// Font variant - use arrow and indentation
		indent := strings.Repeat(" ", (h.Level-1)*indentSize)
		itemText := fmt.Sprintf("%s↳ %s", indent, h.Name)
		result.WriteString(h.Style.Render(itemText))
		result.WriteString("\n")
	}

	// Render children if expanded
	if h.Expanded && len(h.Children) > 0 {
		for _, child := range h.Children {
			result.WriteString(child.Render(indentSize))
		}
	}

	return result.String()
}

// Render renders the hierarchy model
func (m HierarchyModel) Render() string {
	var result strings.Builder

	if m.Title != "" {
		result.WriteString(ui.PageTitle.Render(m.Title))
		result.WriteString("\n\n")
	}

	for i, item := range m.Items {
		result.WriteString(item.Render(m.IndentSize))
		// Add space between font families (level 0 items)
		if i < len(m.Items)-1 && item.Level == 0 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// SetWidth sets the width for the hierarchy model
func (m *HierarchyModel) SetWidth(width int) {
	m.Width = width
}

// SetShowDetails sets whether to show detailed information
func (m *HierarchyModel) SetShowDetails(showDetails bool) {
	m.ShowDetails = showDetails
}

// ToggleItem toggles the expanded state of an item
func (m *HierarchyModel) ToggleItem(itemName string) {
	m.toggleItemRecursive(m.Items, itemName)
}

// toggleItemRecursive recursively toggles an item
func (m *HierarchyModel) toggleItemRecursive(items []HierarchyItem, itemName string) bool {
	for i := range items {
		if items[i].Name == itemName {
			items[i].Expanded = !items[i].Expanded
			return true
		}
		if m.toggleItemRecursive(items[i].Children, itemName) {
			return true
		}
	}
	return false
}

// getDefaultItemStyle returns the default style for hierarchy items
func getDefaultItemStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#f5c2e7")). // Pink for font family names
		Bold(true)
}

// getChildItemStyle returns the style for child items based on level
func getChildItemStyle(level int) lipgloss.Style {
	baseStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6adc8")). // Regular text color for variants
		Bold(false)

	// Add slight indentation styling for deeper levels
	if level > 1 {
		baseStyle = baseStyle.Italic(true)
	}

	return baseStyle
}

// FontFamilyItem creates a font family hierarchy item
func FontFamilyItem(familyName string, variants []string) HierarchyItem {
	item := NewHierarchyItem(familyName)
	item.Style = getFontFamilyStyle()

	// Add variants as children
	for _, variant := range variants {
		child := NewHierarchyItem(variant)
		child.Style = getFontVariantStyle()
		item.AddChild(child)
	}

	return item
}

// getFontFamilyStyle returns the style for font family names
func getFontFamilyStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#f5c2e7")). // Pink
		Bold(true)
}

// getFontVariantStyle returns the style for font variants
func getFontVariantStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6adc8")). // Regular text color
		Bold(false)
}

// CreateFontHierarchy creates a hierarchy from font data
func CreateFontHierarchy(fonts map[string][]string) []HierarchyItem {
	var items []HierarchyItem

	for familyName, variants := range fonts {
		item := FontFamilyItem(familyName, variants)
		items = append(items, item)
	}

	return items
}

// RenderHierarchy renders a hierarchy with the given title and items
func RenderHierarchy(title string, items []HierarchyItem, showDetails bool) string {
	model := NewHierarchyModel(title, items)
	model.SetShowDetails(showDetails)
	return model.Render()
}

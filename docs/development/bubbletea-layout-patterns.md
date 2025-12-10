# Bubble Tea Layout Patterns

## Split View / Side-by-Side Layouts

### Common Patterns in Bubble Tea Applications

#### Pattern 1: Simple Horizontal Split with lipgloss.JoinHorizontal

**Most Common Approach:**
```go
func (m model) View() string {
    // Calculate widths accounting for separator
    separatorWidth := 1
    availableWidth := m.width - separatorWidth
    leftWidth := availableWidth / 2
    rightWidth := availableWidth - leftWidth
    
    // Build each panel
    leftPanel := m.renderLeftPanel(leftWidth)
    rightPanel := m.renderRightPanel(rightWidth)
    
    // Join horizontally with separator
    separator := lipgloss.NewStyle().Render("│")
    combined := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, separator, rightPanel)
    
    return combined
}
```

**Key Points:**
- Use `lipgloss.JoinHorizontal(lipgloss.Top, ...)` for top-aligned panels
- Use `lipgloss.JoinHorizontal(lipgloss.Center, ...)` for center-aligned panels
- Use `lipgloss.JoinHorizontal(lipgloss.Bottom, ...)` for bottom-aligned panels
- Always account for separator width in calculations

#### Pattern 2: Equal Height Panels

**Problem:** Panels with different content heights don't align properly.

**Solution:** Set explicit height on both panels:
```go
func (m model) renderPanel(width, height int, content string) string {
    panelStyle := lipgloss.NewStyle().
        Width(width).
        Height(height).  // Explicit height ensures alignment
        Padding(1, 2)
    
    return panelStyle.Render(content)
}
```

#### Pattern 3: Bordered Panels

**Common Approach:**
```go
func (m model) renderBorderedPanel(width, height int, content string) string {
    // Create border style
    borderStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderTop(true).
        BorderBottom(true).
        BorderLeft(true).
        BorderRight(true).
        Width(width).
        Height(height).
        Padding(1, 2)
    
    return borderStyle.Render(content)
}
```

**Important:** When using borders, account for border width (2 chars) in your calculations:
```go
// Border takes 2 chars (left + right), padding takes 4 chars (2 each side)
contentWidth := width - 2 - 4  // width - border - padding
```

#### Pattern 4: Using Viewport for Scrollable Content

**For scrollable panels:**
```go
import "github.com/charmbracelet/bubbles/viewport"

type model struct {
    leftViewport  viewport.Model
    rightViewport viewport.Model
    width         int
    height        int
}

func (m model) Init() tea.Cmd {
    m.leftViewport.Width = m.width / 2
    m.leftViewport.Height = m.height
    m.rightViewport.Width = m.width / 2
    m.rightViewport.Height = m.height
    return nil
}

func (m model) View() string {
    left := m.leftViewport.View()
    right := m.rightViewport.View()
    return lipgloss.JoinHorizontal(lipgloss.Top, left, "│", right)
}
```

### Best Practices

1. **Width Calculations:**
   ```go
   // Always account for:
   // - Separator width (1 char)
   // - Border width (2 chars if using borders)
   // - Padding (4 chars if padding is 2 on each side)
   separatorWidth := 1
   borderWidth := 2  // if using borders
   paddingWidth := 4  // if padding is 2 on each side
   availableWidth := m.width - separatorWidth - borderWidth - paddingWidth
   ```

2. **Height Alignment:**
   ```go
   // Set explicit height on both panels for proper alignment
   panelHeight := m.height - headerHeight - footerHeight
   leftPanel := lipgloss.NewStyle().Height(panelHeight).Render(leftContent)
   rightPanel := lipgloss.NewStyle().Height(panelHeight).Render(rightContent)
   ```

3. **Content Width Constraints:**
   ```go
   // Constrain content width inside bordered panels
   contentWidth := panelWidth - borderWidth - paddingWidth
   constrainedContent := lipgloss.NewStyle().Width(contentWidth).Render(content)
   ```

4. **Separator Styling:**
   ```go
   // Style separator for better visual separation
   separator := lipgloss.NewStyle().
       Foreground(lipgloss.Color("#44475a")).  // Subtle color
       Render("│")
   ```

### Common Issues and Solutions

#### Issue 1: Panels Not Aligning
**Problem:** Panels have different heights, causing misalignment.

**Solution:** Set explicit height on both panels:
```go
height := m.height - 4  // Account for header/footer
leftPanel := lipgloss.NewStyle().Height(height).Render(leftContent)
rightPanel := lipgloss.NewStyle().Height(height).Render(rightContent)
```

#### Issue 2: Content Overflowing Borders
**Problem:** Content extends beyond panel borders.

**Solution:** Constrain content width:
```go
// Calculate available width inside border
contentWidth := panelWidth - 2  // Border takes 2 chars
content := lipgloss.NewStyle().Width(contentWidth).Render(rawContent)
```

#### Issue 3: Uneven Panel Widths
**Problem:** Panels don't split evenly.

**Solution:** Use integer division and handle remainder:
```go
availableWidth := m.width - separatorWidth
leftWidth := availableWidth / 2
rightWidth := availableWidth - leftWidth  // Handles odd widths
```

### Example: Complete Split View Implementation

```go
package main

import (
    "fmt"
    "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type model struct {
    leftContent  string
    rightContent string
    width        int
    height       int
}

func (m model) Init() tea.Cmd {
    return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        return m, nil
    }
    return m, nil
}

func (m model) View() string {
    // Calculate layout
    separatorWidth := 1
    headerHeight := 2
    footerHeight := 2
    availableWidth := m.width - separatorWidth
    availableHeight := m.height - headerHeight - footerHeight
    
    leftWidth := availableWidth / 2
    rightWidth := availableWidth - leftWidth
    
    // Build panels with borders
    leftPanel := m.renderPanel(leftWidth, availableHeight, m.leftContent)
    rightPanel := m.renderPanel(rightWidth, availableHeight, m.rightContent)
    
    // Create separator
    separator := lipgloss.NewStyle().
        Foreground(lipgloss.Color("#44475a")).
        Render("│")
    
    // Join panels
    combined := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, separator, rightPanel)
    
    // Add header and footer
    header := lipgloss.NewStyle().Bold(true).Render("Header")
    footer := lipgloss.NewStyle().Render("Footer")
    
    return lipgloss.JoinVertical(lipgloss.Left, header, combined, footer)
}

func (m model) renderPanel(width, height int, content string) string {
    // Border style
    borderStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderTop(true).
        BorderBottom(true).
        BorderLeft(true).
        BorderRight(true).
        Width(width).
        Height(height).
        Padding(1, 2)
    
    // Constrain content width (account for border and padding)
    contentWidth := width - 2 - 4  // width - border(2) - padding(4)
    constrainedContent := lipgloss.NewStyle().
        Width(contentWidth).
        Height(height - 2 - 2).  // height - border(2) - padding(2)
        Render(content)
    
    return borderStyle.Render(constrainedContent)
}
```

### References

- **Bubble Tea Docs**: https://github.com/charmbracelet/bubbletea
- **Lipgloss Docs**: https://github.com/charmbracelet/lipgloss
- **Bubbles Library**: https://github.com/charmbracelet/bubbles (viewport, list, etc.)
- **Examples**: https://github.com/charmbracelet/bubbletea/tree/master/examples

### Popular Bubble Tea Apps for Reference

1. **Glow** - Markdown reader with split views
2. **Gum** - CLI tool with various layout examples
3. **Bubble Tea Examples** - Official examples in the repo

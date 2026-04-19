package components

import (
	"strings"

	"fontget/internal/ui"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Card represents a bordered card component with integrated title
type Card struct {
	Title             string
	Content           string
	Sections          []CardSection // If non-empty, Render builds body from sections (uniform width + URL handling)
	Width             int
	VerticalPadding   int // Padding at top and bottom (0 = no padding, 1 = minimal padding)
	HorizontalPadding int // Padding at left and right (0 = no padding, 1 = minimal padding)
}

// CardSection represents a section within a card
type CardSection struct {
	Label string
	Value string
	IsURL bool // When true, Value is rendered as a terminal hyperlink with URL-aware wrapping
}

// CardModel represents a collection of cards
type CardModel struct {
	Title string
	Cards []Card
	Width int
}

// NewCard creates a new card
func NewCard(title, content string) Card {
	return Card{
		Title:             title,
		Content:           content,
		Width:             80,
		VerticalPadding:   1,
		HorizontalPadding: 2,
	}
}

// NewCardWithSections creates a new card with sections
func NewCardWithSections(title string, sections []CardSection) Card {
	return Card{
		Title:             title,
		Sections:          sections,
		Width:             80,
		VerticalPadding:   1,
		HorizontalPadding: 2,
	}
}

// NewCardModel creates a new card model
func NewCardModel(title string, cards []Card) *CardModel {
	return &CardModel{
		Title: title,
		Cards: cards,
		Width: 80,
	}
}

func fitTitlePlainForIntegratedBorder(totalWidth int, titlePlain string) string {
	if titlePlain == "" {
		return ""
	}
	maxSection := totalWidth - 2
	if maxSection < 1 {
		return ansi.Truncate(titlePlain, 1, "...")
	}
	if lipgloss.Width("─"+" "+titlePlain+" "+"─") <= maxSection {
		return titlePlain
	}
	maxW := ansi.StringWidth(titlePlain)
	if maxW > maxSection {
		maxW = maxSection
	}
	for w := maxW; w >= 1; w-- {
		tt := ansi.Truncate(titlePlain, w, "...")
		if lipgloss.Width("─"+" "+tt+" "+"─") <= maxSection {
			return tt
		}
	}
	return ansi.Truncate(titlePlain, 1, "...")
}

// IntegratedRoundedTopBorderLine renders the ╭ ─ Title ─ ─╮ top rule with the same
// styling as Card (CardTitle + border color). titlePlain is measured in plain cells;
// it is truncated so the line never exceeds totalWidth.
func IntegratedRoundedTopBorderLine(totalWidth int, titlePlain string) string {
	if totalWidth <= 0 {
		totalWidth = 80
	}
	titlePlain = fitTitlePlainForIntegratedBorder(totalWidth, strings.TrimSpace(titlePlain))
	styledTitle := ui.CardTitle.Render(titlePlain)

	titleSectionWidth := lipgloss.Width("─" + " " + titlePlain + " " + "─")
	rightWidth := totalWidth - 2 - titleSectionWidth
	if rightWidth < 0 {
		rightWidth = 0
	}

	topLeft := "╭"
	topRight := strings.Repeat("─", rightWidth) + "╮"

	var borderColor lipgloss.TerminalColor
	if ui.CardBorderColorStr != "" {
		borderColor = lipgloss.Color(ui.CardBorderColorStr)
	} else {
		colors := ui.GetCurrentColors()
		if colors != nil && colors.Placeholders != "" {
			borderColor = lipgloss.Color(colors.Placeholders)
		} else {
			borderColor = lipgloss.NoColor{}
		}
	}

	styledTopLeft := lipgloss.NewStyle().Foreground(borderColor).Render(topLeft)
	styledTopRight := lipgloss.NewStyle().Foreground(borderColor).Render(topRight)
	dashStyle := lipgloss.NewStyle().Foreground(borderColor)
	styledDashes := dashStyle.Render("─")
	styledTitleSection := styledDashes + " " + styledTitle + " " + styledDashes

	return styledTopLeft + styledTitleSection + styledTopRight
}

// IntegratedRoundedTopBorderLineError is like IntegratedRoundedTopBorderLine but uses ErrorText for the title
// segment (e.g. result dialogs titled "Error").
func IntegratedRoundedTopBorderLineError(totalWidth int, titlePlain string) string {
	if totalWidth <= 0 {
		totalWidth = 80
	}
	titlePlain = fitTitlePlainForIntegratedBorder(totalWidth, strings.TrimSpace(titlePlain))
	styledTitle := ui.ErrorText.Render(titlePlain)

	titleSectionWidth := lipgloss.Width("─" + " " + titlePlain + " " + "─")
	rightWidth := totalWidth - 2 - titleSectionWidth
	if rightWidth < 0 {
		rightWidth = 0
	}

	topLeft := "╭"
	topRight := strings.Repeat("─", rightWidth) + "╮"

	var borderColor lipgloss.TerminalColor
	if ui.CardBorderColorStr != "" {
		borderColor = lipgloss.Color(ui.CardBorderColorStr)
	} else {
		colors := ui.GetCurrentColors()
		if colors != nil && colors.Placeholders != "" {
			borderColor = lipgloss.Color(colors.Placeholders)
		} else {
			borderColor = lipgloss.NoColor{}
		}
	}

	styledTopLeft := lipgloss.NewStyle().Foreground(borderColor).Render(topLeft)
	styledTopRight := lipgloss.NewStyle().Foreground(borderColor).Render(topRight)
	dashStyle := lipgloss.NewStyle().Foreground(borderColor)
	styledDashes := dashStyle.Render("─")
	styledTitleSection := styledDashes + " " + styledTitle + " " + styledDashes

	return styledTopLeft + styledTitleSection + styledTopRight
}

// Render renders a single card with integrated title
func (c Card) Render() string {
	// Create the content with proper padding
	contentStyle := ui.CardBorder
	if c.VerticalPadding > 0 || c.HorizontalPadding > 0 {
		contentStyle = contentStyle.Padding(c.VerticalPadding, c.HorizontalPadding)
	}

	body := c.Content
	if len(c.Sections) > 0 {
		inner := CardInnerContentWidth(c.Width, c.HorizontalPadding)
		body = buildCardSectionContent(c.Sections, inner)
	}

	// Render the content
	content := contentStyle.Width(c.Width).Render(body)

	// Split content into lines
	lines := strings.Split(content, "\n")
	if len(lines) < 2 {
		return content
	}

	// Use the bottom border line's width so the custom top border matches exactly
	lastLine := lines[len(lines)-1]
	totalWidth := lipgloss.Width(lastLine)
	if totalWidth <= 0 {
		totalWidth = c.Width
	}
	if totalWidth <= 0 {
		totalWidth = 80
	}

	titleLine := IntegratedRoundedTopBorderLine(totalWidth, c.Title)

	// Reconstruct the content with integrated title
	var result strings.Builder
	result.WriteString(titleLine)
	result.WriteString("\n")

	for i := 1; i < len(lines); i++ {
		result.WriteString(lines[i])
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// Render renders the card model
func (m CardModel) Render() string {
	var result strings.Builder

	if m.Title != "" {
		result.WriteString(ui.PageTitle.Render(m.Title))
		result.WriteString("\n\n")
	}

	for i, card := range m.Cards {
		card.Width = m.Width
		result.WriteString(card.Render())
		if i < len(m.Cards)-1 {
			result.WriteString("\n\n")
		}
	}

	return result.String()
}

// SetWidth sets the width for all cards
func (m *CardModel) SetWidth(width int) {
	m.Width = width
	for i := range m.Cards {
		m.Cards[i].Width = width
	}
}

// AddCard adds a card to the model
func (m *CardModel) AddCard(card Card) {
	m.Cards = append(m.Cards, card)
}

// FontDetailsCard creates a comprehensive font details card with tags and metadata
func FontDetailsCard(name, id, category, tags, lastModified, sourceURL, popularity string) Card {
	sections := []CardSection{
		{Label: "Name", Value: name},
		{Label: "ID", Value: id},
	}

	// Add source URL under ID
	if sourceURL != "" {
		sections = append(sections, CardSection{Label: "Source URL", Value: sourceURL, IsURL: true})
	}

	// Add spacing before category section
	sections = append(sections, CardSection{Label: "", Value: ""}) // Empty line

	// Add category, tags, and popularity
	sections = append(sections, CardSection{Label: "Category", Value: category})

	if tags != "" {
		sections = append(sections, CardSection{Label: "Tags", Value: tags})
	}
	if popularity != "" {
		sections = append(sections, CardSection{Label: "Popularity", Value: popularity})
	}

	return NewCardWithSections("Font Details", sections)
}

// LicenseInfoCard creates a card for license information
func LicenseInfoCard(license, url string) Card {
	sections := []CardSection{
		{Label: "License", Value: license},
	}

	if url != "" {
		sections = append(sections, CardSection{
			Label: "URL",
			Value: url,
			IsURL: true,
		})
	}

	return NewCardWithSections("License Information", sections)
}

// CustomCard creates a custom card with the given title and content
func CustomCard(title, content string) Card {
	return NewCard(title, content)
}

// CustomCardWithSections creates a custom card with sections
func CustomCardWithSections(title string, sections []CardSection) Card {
	return NewCardWithSections(title, sections)
}

// ConfigurationInfoCard creates a card for configuration information
func ConfigurationInfoCard(configPath, editor, usePopularitySort string) Card {
	sections := []CardSection{
		{Label: "Location", Value: configPath},
		{Label: "Default Editor", Value: editor},
		{Label: "Use Popularity Sort", Value: usePopularitySort},
	}

	return NewCardWithSections("Configuration Information", sections)
}

// LoggingConfigCard creates a card for logging configuration
func LoggingConfigCard(logPath, maxSize, maxFiles string) Card {
	sections := []CardSection{
		{Label: "Log Path", Value: logPath},
		{Label: "Max Log Size", Value: maxSize},
		{Label: "Max Log Files", Value: maxFiles},
	}

	return NewCardWithSections("Log Settings", sections)
}

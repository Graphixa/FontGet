package components

import (
	"strings"

	"fontget/internal/ui"

	"github.com/charmbracelet/lipgloss"
)

// Card represents a bordered card component with integrated title
type Card struct {
	Title             string
	Content           string
	Width             int
	VerticalPadding   int // Padding at top and bottom (0 = no padding, 1 = minimal padding)
	HorizontalPadding int // Padding at left and right (0 = no padding, 1 = minimal padding)
}

// CardSection represents a section within a card
type CardSection struct {
	Label string
	Value string
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
	var content strings.Builder

	for i, section := range sections {
		if section.Label != "" {
			content.WriteString(ui.CardLabel.Render(section.Label + ": "))
		}
		content.WriteString(ui.CardContent.Render(section.Value))
		if i < len(sections)-1 {
			content.WriteString("\n")
		}
	}

	return Card{
		Title:             title,
		Content:           content.String(),
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

// Render renders a single card with integrated title
func (c Card) Render() string {
	// Create the styled title
	titleText := c.Title
	styledTitle := ui.CardTitle.Render(titleText)

	// Create the content with proper padding
	contentStyle := ui.CardBorder
	if c.VerticalPadding > 0 || c.HorizontalPadding > 0 {
		contentStyle = contentStyle.Padding(c.VerticalPadding, c.HorizontalPadding)
	}

	// Render the content
	content := contentStyle.Width(c.Width).Render(c.Content)

	// Split content into lines
	lines := strings.Split(content, "\n")
	if len(lines) < 2 {
		return content
	}

	// Manually construct the top border with integrated title
	// Use a reasonable fixed width instead of the original border width
	totalWidth := c.Width
	if totalWidth <= 0 {
		totalWidth = 80 // Default width
	}

	// Calculate the length of the title (without ANSI codes)
	// Account for the CardTitle padding (0, 1) which adds 2 characters total
	plainTitleLength := len(c.Title) + 2

	// Calculate the remaining width for the right side
	// Total width - left corner (1) - space (1) - title length (including padding) - space (1) - right corner (1)
	rightWidth := totalWidth - 1 - 1 - plainTitleLength - 1 - 1

	// Ensure we don't have negative width
	if rightWidth < 0 {
		rightWidth = 0
	}

	// Create the integrated title line with proper border styling
	// Use the same border color as CardBorder (Overlay 1: #7f849c)
	borderColor := "#7f849c"
	topLeft := "╭"
	topRight := strings.Repeat("─", rightWidth) + "╮"

	// Apply border color to the top border elements
	styledTopLeft := lipgloss.NewStyle().Foreground(lipgloss.Color(borderColor)).Render(topLeft)
	styledTopRight := lipgloss.NewStyle().Foreground(lipgloss.Color(borderColor)).Render(topRight)

	// Create dashes with border color
	dashStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(borderColor))
	styledDashes := dashStyle.Render("─")

	// Reconstruct the title section with styled dashes
	styledTitleSection := styledDashes + " " + styledTitle + " " + styledDashes

	titleLine := styledTopLeft + styledTitleSection + styledTopRight

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

// FontDetailsCard creates a card for font details
func FontDetailsCard(name, id, category string) Card {
	sections := []CardSection{
		{Label: "Name", Value: name},
		{Label: "ID", Value: id},
		{Label: "Category", Value: category},
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
		})
	}

	return NewCardWithSections("License Information", sections)
}

// AvailableFilesCard creates a card for available files
func AvailableFilesCard(files []string) Card {
	if len(files) == 0 {
		return NewCard("Available Files", "No files available")
	}

	var content strings.Builder
	for i, file := range files {
		// Use a simple bullet point without extra spaces
		content.WriteString("• " + file)
		if i < len(files)-1 {
			content.WriteString("\n")
		}
	}

	// Create a custom card that handles URLs better
	card := Card{
		Title:             "Available Files",
		Content:           content.String(),
		Width:             80,
		VerticalPadding:   1,
		HorizontalPadding: 0, // No horizontal padding for URLs
	}

	return card
}

// MetadataCard creates a card for metadata
func MetadataCard(lastModified, sourceURL, popularity string) Card {
	sections := []CardSection{
		{Label: "Last Modified", Value: lastModified},
		{Label: "Source URL", Value: sourceURL},
	}

	if popularity != "" {
		sections = append(sections, CardSection{
			Label: "Popularity",
			Value: popularity,
		})
	}

	return NewCardWithSections("Metadata", sections)
}

// SourceInfoCard creates a card for source information
func SourceInfoCard(sourceName, sourceURL, description string) Card {
	sections := []CardSection{
		{Label: "Source", Value: sourceName},
		{Label: "URL", Value: sourceURL},
	}

	// Only add description if it's provided
	if description != "" {
		sections = append(sections, CardSection{
			Label: "Description",
			Value: description,
		})
	}

	return NewCardWithSections("Source Information", sections)
}

// CustomCard creates a custom card with the given title and content
func CustomCard(title, content string) Card {
	return NewCard(title, content)
}

// CustomCardWithSections creates a custom card with sections
func CustomCardWithSections(title string, sections []CardSection) Card {
	return NewCardWithSections(title, sections)
}

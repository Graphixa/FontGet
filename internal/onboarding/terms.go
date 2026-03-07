package onboarding

import (
	_ "embed"
	"strings"

	"fontget/internal/ui"

	"gopkg.in/yaml.v3"
)

//go:embed terms_of_use.yaml
var termsOfUseYAML []byte

// Section is one block on the Terms of Use screen: name (optional), style (UI style key), and content or items.
type Section struct {
	Name    string   `yaml:"name"`
	Style   string   `yaml:"style"`
	Content string   `yaml:"content"`
	Items   []string `yaml:"items"`
}

type termsOfUseFile struct {
	Sections []Section `yaml:"sections"`
}

var termsSections []Section

// styleRegistry maps style keys from the YAML to ui render functions. Wrappers needed because lipgloss Render is func(...string) string.
var styleRegistry = map[string]func(string) string{
	"PageTitle":  func(s string) string { return ui.PageTitle.Render(s) },
	"Text":       func(s string) string { return ui.Text.Render(s) },
	"InfoText":   func(s string) string { return ui.InfoText.Render(s) },
	"SourceName": func(s string) string { return ui.TableSourceName.Render(s) },
}

func init() {
	var file termsOfUseFile
	if err := yaml.Unmarshal(termsOfUseYAML, &file); err != nil {
		panic("terms_of_use.yaml: " + err.Error())
	}
	if len(file.Sections) == 0 {
		panic("terms_of_use.yaml: must define at least one section")
	}
	for i := range file.Sections {
		s := &file.Sections[i]
		s.Name = strings.TrimSpace(s.Name)
		s.Style = strings.TrimSpace(s.Style)
		s.Content = strings.TrimSpace(s.Content)
		for j := range s.Items {
			s.Items[j] = strings.TrimSpace(s.Items[j])
		}
	}
	termsSections = file.Sections
}

// TermsOfUseSections returns the ordered list of sections from the terms file (or defaults).
func TermsOfUseSections() []Section {
	return termsSections
}

// StyleRenderer returns a function that renders a string with the named style (e.g. "PageTitle", "Text"). Unknown names fall back to Text.
func StyleRenderer(styleName string) func(string) string {
	if fn, ok := styleRegistry[styleName]; ok {
		return fn
	}
	return func(s string) string { return ui.Text.Render(s) }
}

// TermsOfUseTitle returns the content of the first section named "Title" (for backward compatibility).
func TermsOfUseTitle() string {
	for _, s := range termsSections {
		if s.Name == "Title" && s.Content != "" {
			return s.Content
		}
	}
	return "Terms of Use"
}

// TermsOfUseIntro returns the content of the first section named "Intro".
func TermsOfUseIntro() string {
	for _, s := range termsSections {
		if s.Name == "Intro" {
			return s.Content
		}
	}
	return ""
}

// TermsOfUseSources returns the items of the first section named "Sources".
func TermsOfUseSources() []string {
	for _, s := range termsSections {
		if s.Name == "Sources" && len(s.Items) > 0 {
			return s.Items
		}
	}
	return nil
}

// TermsOfUseTerms returns the content of the first section named "Terms" or "Disclaimer" (for license prompt).
func TermsOfUseTerms() string {
	for _, s := range termsSections {
		if s.Name == "Terms" || s.Name == "Disclaimer" {
			return s.Content
		}
	}
	return ""
}

// TermsOfUseAcceptanceLine returns the content of the first section named "Acceptance".
func TermsOfUseAcceptanceLine() string {
	for _, s := range termsSections {
		if s.Name == "Acceptance" {
			return s.Content
		}
	}
	return ""
}

// TermsOfUseIntroText returns the main terms body (for license prompt; backward compatibility).
func TermsOfUseIntroText() string {
	return TermsOfUseTerms()
}

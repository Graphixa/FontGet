package cmd

import (
	"fmt"
	"sort"
	"strings"

	"fontget/internal/config"
)

// SourceItem represents a source for sorting purposes
type SourceItem struct {
	Name      string
	Prefix    string
	URL       string
	Enabled   bool
	IsBuiltIn bool
	Priority  int
}

// SortSources sorts sources by type (built-in first) then by priority order
func SortSources(sources []SourceItem) {
	sort.Slice(sources, func(i, j int) bool {
		// Built-in sources come first
		if sources[i].IsBuiltIn != sources[j].IsBuiltIn {
			return sources[i].IsBuiltIn
		}
		// Within same type, sort by priority (lower number = higher priority)
		if sources[i].Priority != sources[j].Priority {
			return sources[i].Priority < sources[j].Priority
		}
		// If priorities are equal, sort by name
		return sources[i].Name < sources[j].Name
	})
}

// GetEnabledSourcesInOrder returns enabled sources in priority order from config manifest
func GetEnabledSourcesInOrder(manifest *config.Manifest) []string {
	var sources []SourceItem

	for name, source := range manifest.Sources {
		if source.Enabled {
			sources = append(sources, SourceItem{
				Name:     name,
				Priority: source.Priority,
			})
		}
	}

	// Sort by priority
	SortSources(sources)

	var result []string
	for _, source := range sources {
		result = append(result, source.Name)
	}
	return result
}

// FindSourceIndex finds the index of a source by name
func FindSourceIndex(sources []SourceItem, name string) int {
	for i, source := range sources {
		if source.Name == name {
			return i
		}
	}
	return -1
}

// Input validation constants
const (
	MinInputWidth     = 30
	MaxInputWidth     = 100
	DefaultInputWidth = 50
)

// ValidationError represents a validation error with field context
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationResult represents the result of form validation
type ValidationResult struct {
	IsValid bool
	Errors  []ValidationError
}

// AddError adds an error to the validation result
func (vr *ValidationResult) AddError(field, message string) {
	vr.IsValid = false
	vr.Errors = append(vr.Errors, ValidationError{Field: field, Message: message})
}

// GetFirstError returns the first error message, or empty string if no errors
func (vr *ValidationResult) GetFirstError() string {
	if len(vr.Errors) == 0 {
		return ""
	}
	return vr.Errors[0].Message
}

// ValidateRequired validates that a field is not empty
func ValidateRequired(value, fieldName string) error {
	if strings.TrimSpace(value) == "" {
		return ValidationError{Field: fieldName, Message: fieldName + " is required"}
	}
	return nil
}

// ValidateURL validates that a string looks like a URL
func ValidateURL(url string) error {
	url = strings.TrimSpace(url)
	if url == "" {
		return ValidationError{Field: "URL", Message: "URL is required"}
	}

	// Basic URL validation - must start with http:// or https://
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return ValidationError{Field: "URL", Message: "URL must start with http:// or https://"}
	}

	return nil
}

// ValidatePrefix validates that a prefix is valid
func ValidatePrefix(prefix string) error {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return nil // Prefix is optional, will be auto-generated
	}

	// Prefix should be lowercase and alphanumeric
	if strings.ToLower(prefix) != prefix {
		return ValidationError{Field: "Prefix", Message: "Prefix must be lowercase"}
	}

	// Check for valid characters (alphanumeric and hyphens)
	for _, char := range prefix {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-') {
			return ValidationError{Field: "Prefix", Message: "Prefix can only contain lowercase letters, numbers, and hyphens"}
		}
	}

	return nil
}

// ValidateSourceForm validates a complete source form
func ValidateSourceForm(name, url, prefix string, existingSources []SourceItem, editingIndex int) ValidationResult {
	result := ValidationResult{IsValid: true, Errors: []ValidationError{}}

	// Validate name
	if err := ValidateRequired(name, "Name"); err != nil {
		if validationErr, ok := err.(ValidationError); ok {
			result.AddError("Name", validationErr.Message)
		} else {
			result.AddError("Name", err.Error())
		}
	}

	// Validate URL
	if err := ValidateURL(url); err != nil {
		if validationErr, ok := err.(ValidationError); ok {
			result.AddError("URL", validationErr.Message)
		} else {
			result.AddError("URL", err.Error())
		}
	}

	// Validate prefix
	if err := ValidatePrefix(prefix); err != nil {
		if validationErr, ok := err.(ValidationError); ok {
			result.AddError("Prefix", validationErr.Message)
		} else {
			result.AddError("Prefix", err.Error())
		}
	}

	// Check for duplicate names (except when editing the same source)
	for i, source := range existingSources {
		if source.Name == name && (editingIndex == -1 || i != editingIndex) {
			result.AddError("Name", "Source with this name already exists")
			break
		}
	}

	// Check for duplicate URLs (except when editing the same source)
	for i, source := range existingSources {
		if source.URL == url && (editingIndex == -1 || i != editingIndex) {
			result.AddError("URL", "Source with this URL already exists")
			break
		}
	}

	// Check for duplicate prefixes (except when editing the same source)
	if prefix != "" {
		for i, source := range existingSources {
			if source.Prefix == prefix && (editingIndex == -1 || i != editingIndex) {
				result.AddError("Prefix", "Source with this prefix already exists")
				break
			}
		}
	}

	return result
}

// AutoGeneratePrefix generates a prefix from a source name
func AutoGeneratePrefix(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}

// CalculateInputWidth calculates appropriate input width based on terminal width
func CalculateInputWidth(terminalWidth int) int {
	// Account for: "> " (2) + "Name: " (6) + " " (1) + some padding = ~10
	availableWidth := terminalWidth - 10
	if availableWidth < MinInputWidth {
		return MinInputWidth
	}
	if availableWidth > MaxInputWidth {
		return MaxInputWidth
	}
	return availableWidth
}

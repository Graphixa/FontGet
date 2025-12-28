package config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// ValidationError represents a single validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}

	var messages []string
	for _, err := range e {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "\n  - ")
}

// ValidateStrictAppConfig performs strict validation on app config with multi-error collection
func ValidateStrictAppConfig(data map[string]interface{}) error {
	var errors ValidationErrors

	// Validate ConfigVersion (optional for backward compatibility - old configs may not have it)
	if configVersion, exists := data["ConfigVersion"]; exists {
		if versionStr, ok := configVersion.(string); ok {
			if versionStr != "" && versionStr != "1.0" && versionStr != "2.0" {
				errors = append(errors, ValidationError{
					Field:   "ConfigVersion",
					Message: fmt.Sprintf("unknown config version '%s' (supported: 1.0, 2.0)", versionStr),
				})
			}
		} else {
			errors = append(errors, ValidationError{
				Field:   "ConfigVersion",
				Message: fmt.Sprintf("must be a string, got %s", getTypeName(configVersion)),
			})
		}
	}
	// ConfigVersion is optional - old configs without it will be migrated

	// Validate Configuration section
	if configSection, exists := data["Configuration"]; exists {
		if configMap, ok := configSection.(map[string]interface{}); ok {
			errors = append(errors, validateConfigurationSection(configMap)...)
		} else {
			errors = append(errors, ValidationError{
				Field:   "Configuration",
				Message: "must be an object",
			})
		}
	} else {
		errors = append(errors, ValidationError{
			Field:   "Configuration",
			Message: "section is required",
		})
	}

	// Validate Logging section
	if loggingSection, exists := data["Logging"]; exists {
		if loggingMap, ok := loggingSection.(map[string]interface{}); ok {
			errors = append(errors, validateLoggingSection(loggingMap)...)
		} else {
			errors = append(errors, ValidationError{
				Field:   "Logging",
				Message: "must be an object",
			})
		}
	} else {
		errors = append(errors, ValidationError{
			Field:   "Logging",
			Message: "section is required",
		})
	}

	// Validate Update section (optional for backward compatibility)
	if updateSection, exists := data["Update"]; exists {
		if updateMap, ok := updateSection.(map[string]interface{}); ok {
			errors = append(errors, validateUpdateSection(updateMap)...)
		} else {
			errors = append(errors, ValidationError{
				Field:   "Update",
				Message: "must be an object",
			})
		}
	}
	// Update section is optional, so we don't require it

	// Validate Theme (optional for backward compatibility)
	// Theme can be a string (new format) or a map with Name (old format)
	if themeSection, exists := data["Theme"]; exists {
		errors = append(errors, validateThemeSection(themeSection)...)
	}
	// Theme section is optional, so we don't require it

	if len(errors) > 0 {
		return errors
	}
	return nil
}

// validateConfigurationSection validates the Configuration section
func validateConfigurationSection(config map[string]interface{}) ValidationErrors {
	var errors ValidationErrors

	// Validate DefaultEditor (optional - uses system default if empty)
	if defaultEditor, exists := config["DefaultEditor"]; exists {
		if _, ok := defaultEditor.(string); ok {
			// DefaultEditor is optional, so empty string is valid
			// No validation needed for empty string
		} else {
			errors = append(errors, ValidationError{
				Field:   "Configuration.DefaultEditor",
				Message: fmt.Sprintf("must be a string, got %s", getTypeName(defaultEditor)),
			})
		}
	}
	// If DefaultEditor doesn't exist, that's fine - it's optional

	// Validate EnablePopularitySort (required boolean)
	if usePopularitySort, exists := config["EnablePopularitySort"]; exists {
		switch v := usePopularitySort.(type) {
		case bool:
			// Valid boolean value
		case string:
			// Try to convert string to bool
			switch strings.ToLower(v) {
			case "true", "1", "yes":
				// Valid boolean string
			case "false", "0", "no":
				// Valid boolean string
			default:
				errors = append(errors, ValidationError{
					Field:   "Configuration.EnablePopularitySort",
					Message: fmt.Sprintf("must be a boolean, got string '%s'", v),
				})
			}
		default:
			errors = append(errors, ValidationError{
				Field:   "Configuration.EnablePopularitySort",
				Message: fmt.Sprintf("must be a boolean, got %s", getTypeName(usePopularitySort)),
			})
		}
	} else {
		errors = append(errors, ValidationError{
			Field:   "Configuration.EnablePopularitySort",
			Message: "field is required",
		})
	}

	return errors
}

// validateLoggingSection validates the Logging section
func validateLoggingSection(logging map[string]interface{}) ValidationErrors {
	var errors ValidationErrors

	// Validate LogPath
	if logPath, exists := logging["LogPath"]; exists {
		if logPathStr, ok := logPath.(string); ok {
			if logPathStr == "" {
				errors = append(errors, ValidationError{
					Field:   "Logging.LogPath",
					Message: "cannot be empty",
				})
			}
		} else {
			errors = append(errors, ValidationError{
				Field:   "Logging.LogPath",
				Message: fmt.Sprintf("must be a string, got %s", getTypeName(logPath)),
			})
		}
	} else {
		errors = append(errors, ValidationError{
			Field:   "Logging.LogPath",
			Message: "field is required",
		})
	}

	// Validate MaxLogSize
	if maxSize, exists := logging["MaxLogSize"]; exists {
		if maxSizeStr, ok := maxSize.(string); ok {
			if maxSizeStr == "" {
				errors = append(errors, ValidationError{
					Field:   "Logging.MaxLogSize",
					Message: "cannot be empty",
				})
			}
		} else {
			errors = append(errors, ValidationError{
				Field:   "Logging.MaxLogSize",
				Message: fmt.Sprintf("must be a string, got %s", getTypeName(maxSize)),
			})
		}
	} else {
		errors = append(errors, ValidationError{
			Field:   "Logging.MaxLogSize",
			Message: "field is required",
		})
	}

	// Validate MaxLogFiles
	if maxFiles, exists := logging["MaxLogFiles"]; exists {
		switch v := maxFiles.(type) {
		case int:
			if v <= 0 {
				errors = append(errors, ValidationError{
					Field:   "Logging.MaxLogFiles",
					Message: "must be greater than 0",
				})
			}
		case float64:
			// YAML numbers are often parsed as float64
			if v <= 0 {
				errors = append(errors, ValidationError{
					Field:   "Logging.MaxLogFiles",
					Message: "must be greater than 0",
				})
			}
		case string:
			// Try to convert string to int
			if intVal, err := strconv.Atoi(v); err != nil {
				errors = append(errors, ValidationError{
					Field:   "Logging.MaxLogFiles",
					Message: fmt.Sprintf("must be an integer, got string '%s'", v),
				})
			} else if intVal <= 0 {
				errors = append(errors, ValidationError{
					Field:   "Logging.MaxLogFiles",
					Message: "must be greater than 0",
				})
			}
		default:
			errors = append(errors, ValidationError{
				Field:   "Logging.MaxLogFiles",
				Message: fmt.Sprintf("must be an integer, got %s", getTypeName(maxFiles)),
			})
		}
	} else {
		errors = append(errors, ValidationError{
			Field:   "Logging.MaxLogFiles",
			Message: "field is required",
		})
	}

	return errors
}

// validateUpdateSection validates the Update section
func validateUpdateSection(update map[string]interface{}) ValidationErrors {
	var errors ValidationErrors

	// Validate AutoCheck (optional boolean, defaults to true)
	if autoCheck, exists := update["AutoCheck"]; exists {
		switch v := autoCheck.(type) {
		case bool:
			// Valid boolean value
		case string:
			// Try to convert string to bool
			switch strings.ToLower(v) {
			case "true", "1", "yes":
				// Valid boolean string
			case "false", "0", "no":
				// Valid boolean string
			default:
				errors = append(errors, ValidationError{
					Field:   "Update.AutoCheck",
					Message: fmt.Sprintf("must be a boolean, got string '%s'", v),
				})
			}
		default:
			errors = append(errors, ValidationError{
				Field:   "Update.AutoCheck",
				Message: fmt.Sprintf("must be a boolean, got %s", getTypeName(autoCheck)),
			})
		}
	}

	// Validate AutoUpdate (optional boolean, defaults to false)
	if autoUpdate, exists := update["AutoUpdate"]; exists {
		switch v := autoUpdate.(type) {
		case bool:
			// Valid boolean value
		case string:
			// Try to convert string to bool
			switch strings.ToLower(v) {
			case "true", "1", "yes":
				// Valid boolean string
			case "false", "0", "no":
				// Valid boolean string
			default:
				errors = append(errors, ValidationError{
					Field:   "Update.AutoUpdate",
					Message: fmt.Sprintf("must be a boolean, got string '%s'", v),
				})
			}
		default:
			errors = append(errors, ValidationError{
				Field:   "Update.AutoUpdate",
				Message: fmt.Sprintf("must be a boolean, got %s", getTypeName(autoUpdate)),
			})
		}
	}

	// Validate UpdateCheckInterval (optional integer, must be > 0 if present)
	if checkInterval, exists := update["UpdateCheckInterval"]; exists {
		switch v := checkInterval.(type) {
		case int:
			if v <= 0 {
				errors = append(errors, ValidationError{
					Field:   "Update.UpdateCheckInterval",
					Message: "must be greater than 0",
				})
			}
		case float64:
			// YAML numbers are often parsed as float64
			if v <= 0 {
				errors = append(errors, ValidationError{
					Field:   "Update.UpdateCheckInterval",
					Message: "must be greater than 0",
				})
			}
		case string:
			// Try to convert string to int
			if intVal, err := strconv.Atoi(v); err != nil {
				errors = append(errors, ValidationError{
					Field:   "Update.UpdateCheckInterval",
					Message: fmt.Sprintf("must be an integer, got string '%s'", v),
				})
			} else if intVal <= 0 {
				errors = append(errors, ValidationError{
					Field:   "Update.UpdateCheckInterval",
					Message: "must be greater than 0",
				})
			}
		default:
			errors = append(errors, ValidationError{
				Field:   "Update.UpdateCheckInterval",
				Message: fmt.Sprintf("must be an integer, got %s", getTypeName(checkInterval)),
			})
		}
	}

	// Validate LastUpdateCheck (optional string, should be ISO timestamp if present)
	if lastChecked, exists := update["LastUpdateCheck"]; exists {
		if _, ok := lastChecked.(string); !ok {
			errors = append(errors, ValidationError{
				Field:   "Update.LastUpdateCheck",
				Message: fmt.Sprintf("must be a string, got %s", getTypeName(lastChecked)),
			})
		}
	}

	// Validate UpdateChannel (optional string, should be one of: stable, beta, nightly)
	if updateChannel, exists := update["UpdateChannel"]; exists {
		if channelStr, ok := updateChannel.(string); ok {
			validChannels := []string{"stable", "beta", "nightly"}
			valid := false
			for _, validChannel := range validChannels {
				if channelStr == validChannel {
					valid = true
					break
				}
			}
			if !valid {
				errors = append(errors, ValidationError{
					Field:   "Update.UpdateChannel",
					Message: fmt.Sprintf("must be one of: %s, got '%s'", strings.Join(validChannels, ", "), channelStr),
				})
			}
		} else {
			errors = append(errors, ValidationError{
				Field:   "Update.UpdateChannel",
				Message: fmt.Sprintf("must be a string, got %s", getTypeName(updateChannel)),
			})
		}
	}

	return errors
}

// validateThemeSection validates the Theme value
// Supports both old format (map with Name) and new format (string) for backward compatibility
func validateThemeSection(theme interface{}) ValidationErrors {
	var errors ValidationErrors

	if theme == nil {
		return errors
	}

	// New format: Theme is a string value
	if themeStr, ok := theme.(string); ok {
		// String is valid (can be empty to use system theme)
		_ = themeStr // No validation needed for string values
		return errors
	}

	// Old format: Theme is a map with Name field (backward compatibility)
	if themeMap, ok := theme.(map[string]interface{}); ok {
		// Validate Name field if it exists
		if name, exists := themeMap["Name"]; exists {
			if _, ok := name.(string); !ok {
				errors = append(errors, ValidationError{
					Field:   "Theme.Name",
					Message: fmt.Sprintf("must be a string, got %s", getTypeName(name)),
				})
			}
		}
		// Name is optional, so empty or missing is fine
		return errors
	}

	// Invalid format
	errors = append(errors, ValidationError{
		Field:   "Theme",
		Message: fmt.Sprintf("must be a string or an object with Name field, got %s", getTypeName(theme)),
	})

	return errors
}

// getTypeName returns a human-readable type name
func getTypeName(v interface{}) string {
	if v == nil {
		return "null"
	}
	return reflect.TypeOf(v).String()
}

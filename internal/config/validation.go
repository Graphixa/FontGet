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

// ValidateStrictYAMLConfig performs strict validation on YAML config with multi-error collection
func ValidateStrictYAMLConfig(data map[string]interface{}) error {
	var errors ValidationErrors

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

	if len(errors) > 0 {
		return errors
	}
	return nil
}

// ValidateStrictSourcesConfig performs strict validation on sources config with multi-error collection
func ValidateStrictSourcesConfig(data map[string]interface{}) error {
	var errors ValidationErrors

	// Validate sources section
	if sourcesSection, exists := data["sources"]; exists {
		if sourcesMap, ok := sourcesSection.(map[string]interface{}); ok {
			if len(sourcesMap) == 0 {
				errors = append(errors, ValidationError{
					Field:   "sources",
					Message: "at least one source must be configured",
				})
			} else {
				for name, sourceData := range sourcesMap {
					if sourceMap, ok := sourceData.(map[string]interface{}); ok {
						errors = append(errors, validateSource(name, sourceMap)...)
					} else {
						errors = append(errors, ValidationError{
							Field:   fmt.Sprintf("sources.%s", name),
							Message: "must be an object",
						})
					}
				}
			}
		} else {
			errors = append(errors, ValidationError{
				Field:   "sources",
				Message: "must be an object",
			})
		}
	} else {
		errors = append(errors, ValidationError{
			Field:   "sources",
			Message: "section is required",
		})
	}

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

	// Validate MaxSize
	if maxSize, exists := logging["MaxSize"]; exists {
		if maxSizeStr, ok := maxSize.(string); ok {
			if maxSizeStr == "" {
				errors = append(errors, ValidationError{
					Field:   "Logging.MaxSize",
					Message: "cannot be empty",
				})
			}
		} else {
			errors = append(errors, ValidationError{
				Field:   "Logging.MaxSize",
				Message: fmt.Sprintf("must be a string, got %s", getTypeName(maxSize)),
			})
		}
	} else {
		errors = append(errors, ValidationError{
			Field:   "Logging.MaxSize",
			Message: "field is required",
		})
	}

	// Validate MaxFiles
	if maxFiles, exists := logging["MaxFiles"]; exists {
		switch v := maxFiles.(type) {
		case int:
			if v <= 0 {
				errors = append(errors, ValidationError{
					Field:   "Logging.MaxFiles",
					Message: "must be greater than 0",
				})
			}
		case float64:
			// YAML numbers are often parsed as float64
			if v <= 0 {
				errors = append(errors, ValidationError{
					Field:   "Logging.MaxFiles",
					Message: "must be greater than 0",
				})
			}
		case string:
			// Try to convert string to int
			if intVal, err := strconv.Atoi(v); err != nil {
				errors = append(errors, ValidationError{
					Field:   "Logging.MaxFiles",
					Message: fmt.Sprintf("must be an integer, got string '%s'", v),
				})
			} else if intVal <= 0 {
				errors = append(errors, ValidationError{
					Field:   "Logging.MaxFiles",
					Message: "must be greater than 0",
				})
			}
		default:
			errors = append(errors, ValidationError{
				Field:   "Logging.MaxFiles",
				Message: fmt.Sprintf("must be an integer, got %s", getTypeName(maxFiles)),
			})
		}
	} else {
		errors = append(errors, ValidationError{
			Field:   "Logging.MaxFiles",
			Message: "field is required",
		})
	}

	return errors
}

// validateSource validates a single source
func validateSource(name string, source map[string]interface{}) ValidationErrors {
	var errors ValidationErrors

	// Validate Path
	if path, exists := source["path"]; exists {
		if pathStr, ok := path.(string); ok {
			if pathStr == "" {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("sources.%s.path", name),
					Message: "cannot be empty",
				})
			}
		} else {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("sources.%s.path", name),
				Message: fmt.Sprintf("must be a string, got %s", getTypeName(path)),
			})
		}
	} else {
		errors = append(errors, ValidationError{
			Field:   fmt.Sprintf("sources.%s.path", name),
			Message: "field is required",
		})
	}

	// Validate Prefix
	if prefix, exists := source["prefix"]; exists {
		if prefixStr, ok := prefix.(string); ok {
			if prefixStr == "" {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("sources.%s.prefix", name),
					Message: "cannot be empty",
				})
			}
		} else {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("sources.%s.prefix", name),
				Message: fmt.Sprintf("must be a string, got %s", getTypeName(prefix)),
			})
		}
	} else {
		errors = append(errors, ValidationError{
			Field:   fmt.Sprintf("sources.%s.prefix", name),
			Message: "field is required",
		})
	}

	// Validate Enabled
	if enabled, exists := source["enabled"]; exists {
		switch v := enabled.(type) {
		case bool:
			// Valid boolean value
		case string:
			// Try to convert string to bool
			switch strings.ToLower(v) {
			case "true", "1", "yes", "on":
				// Valid boolean string
			case "false", "0", "no", "off":
				// Valid boolean string
			default:
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("sources.%s.enabled", name),
					Message: fmt.Sprintf("must be a boolean, got string '%s'", v),
				})
			}
		default:
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("sources.%s.enabled", name),
				Message: fmt.Sprintf("must be a boolean, got %s", getTypeName(enabled)),
			})
		}
	} else {
		errors = append(errors, ValidationError{
			Field:   fmt.Sprintf("sources.%s.enabled", name),
			Message: "field is required",
		})
	}

	return errors
}

// getTypeName returns a human-readable type name
func getTypeName(v interface{}) string {
	if v == nil {
		return "null"
	}
	return reflect.TypeOf(v).String()
}

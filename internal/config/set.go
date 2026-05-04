package config

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// excludedKeys lists dotted paths (normalized lowercase) that are not settable.
// All other section.field paths on AppConfig are settable.
var excludedKeys = map[string]bool{
	"version":                true, // top-level schema version
	"update.lastupdatecheck": true,
	"update.nextupdatecheck": true,
}

// getYAMLTagName returns the lowercase name for matching (yaml tag first part, or struct field name).
func getYAMLTagName(f reflect.StructField) string {
	tag := f.Tag.Get("yaml")
	if tag != "" && tag != "-" {
		part := strings.SplitN(tag, ",", 2)[0]
		return strings.ToLower(strings.TrimSpace(part))
	}
	return strings.ToLower(f.Name)
}

// parseBool parses common bool string representations.
func parseBool(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1", "yes":
		return true, nil
	case "false", "0", "no":
		return false, nil
	default:
		return false, fmt.Errorf("invalid bool: %s", s)
	}
}

// parseValueByKind parses s into a reflect.Value of the given kind for setting.
// Supports String, Int, Int32, Int64, Bool. Returns an error for unsupported kinds or parse failure.
func parseValueByKind(s string, k reflect.Kind) (reflect.Value, error) {
	switch k {
	case reflect.String:
		return reflect.ValueOf(s), nil
	case reflect.Int, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid integer: %w", err)
		}
		switch k {
		case reflect.Int:
			return reflect.ValueOf(int(n)), nil
		case reflect.Int32:
			if n > math.MaxInt32 || n < math.MinInt32 {
				return reflect.Value{}, fmt.Errorf("value out of range for int32: %d", n)
			}
			return reflect.ValueOf(int32(n)), nil
		case reflect.Int64:
			return reflect.ValueOf(n), nil
		}
	case reflect.Bool:
		b, err := parseBool(s)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(b), nil
	default:
		return reflect.Value{}, fmt.Errorf("unsupported type %s", k)
	}
	return reflect.Value{}, fmt.Errorf("unsupported type %s", k)
}

// isSettableKind reports whether the kind is one we can set from a string.
func isSettableKind(k reflect.Kind) bool {
	switch k {
	case reflect.String, reflect.Int, reflect.Int32, reflect.Int64, reflect.Bool:
		return true
	}
	return false
}

// findSectionAndField locates the section and field on cfg by dotted key (sectionKey.fieldKey, already normalized).
// Returns the section's reflect.Value, the field index within the section, and the field's Kind.
func findSectionAndField(cfg *AppConfig, sectionKey, fieldKey string) (sectionVal reflect.Value, fieldIndex int, fieldKind reflect.Kind, err error) {
	v := reflect.ValueOf(cfg).Elem()
	t := v.Type()
	if t.Kind() != reflect.Struct {
		return reflect.Value{}, 0, reflect.Invalid, fmt.Errorf("invalid config type")
	}
	for i := 0; i < v.NumField(); i++ {
		sf := t.Field(i)
		name := getYAMLTagName(sf)
		if name != sectionKey {
			continue
		}
		secVal := v.Field(i)
		if secVal.Kind() == reflect.Ptr {
			secVal = secVal.Elem()
		}
		if secVal.Kind() != reflect.Struct {
			return reflect.Value{}, 0, reflect.Invalid, fmt.Errorf("section %s is not a struct", sectionKey)
		}
		secType := secVal.Type()
		for j := 0; j < secVal.NumField(); j++ {
			ff := secType.Field(j)
			fname := getYAMLTagName(ff)
			if fname != fieldKey {
				continue
			}
			fieldKind := secVal.Field(j).Kind()
			if !isSettableKind(fieldKind) {
				return reflect.Value{}, 0, reflect.Invalid, fmt.Errorf("key %s.%s has unsupported type", sectionKey, fieldKey)
			}
			return secVal, j, fieldKind, nil
		}
		return reflect.Value{}, 0, reflect.Invalid, fmt.Errorf("unknown key: %s.%s (use 'fontget config set --help' for valid keys)", sectionKey, fieldKey)
	}
	return reflect.Value{}, 0, reflect.Invalid, fmt.Errorf("unknown key: %s.%s (use 'fontget config set --help' for valid keys)", sectionKey, fieldKey)
}

// SetConfigKey sets a single config key by dotted path (e.g. theme.name, logging.logpath).
// Key is case-insensitive; value is parsed to the correct type from the AppConfig schema.
// Returns an error for unknown keys, excluded keys, unsupported types, or invalid value types.
func SetConfigKey(cfg *AppConfig, key, value string) error {
	normalized := strings.ToLower(strings.TrimSpace(key))
	if excludedKeys[normalized] {
		return fmt.Errorf("key %s is not settable", normalized)
	}
	parts := strings.SplitN(normalized, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid key format: use section.field (e.g. theme.name)")
	}
	sectionKey, fieldKey := parts[0], parts[1]

	sectionVal, fieldIndex, fieldKind, err := findSectionAndField(cfg, sectionKey, fieldKey)
	if err != nil {
		return err
	}

	parsed, err := parseValueByKind(value, fieldKind)
	if err != nil {
		return fmt.Errorf("invalid value for %s: %w", normalized, err)
	}

	fieldVal := sectionVal.Field(fieldIndex)
	if !fieldVal.CanSet() {
		return fmt.Errorf("key %s is not settable", normalized)
	}

	switch fieldKind {
	case reflect.String:
		fieldVal.SetString(parsed.String())
	case reflect.Int, reflect.Int32, reflect.Int64:
		fieldVal.SetInt(parsed.Int())
	case reflect.Bool:
		fieldVal.SetBool(parsed.Bool())
	default:
		return fmt.Errorf("key %s has unsupported type", normalized)
	}
	return nil
}

// SettableKeys returns the list of dotted keys that can be set (lowercase), derived from AppConfig.
// Used for help and documentation. Excluded keys are omitted.
func SettableKeys() []string {
	var keys []string
	v := reflect.ValueOf(&AppConfig{}).Elem()
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		sf := t.Field(i)
		sectionKey := getYAMLTagName(sf)
		// Skip top-level version
		if sectionKey == "version" {
			continue
		}
		secVal := v.Field(i)
		if secVal.Kind() == reflect.Ptr {
			secVal = secVal.Elem()
		}
		if secVal.Kind() != reflect.Struct {
			continue
		}
		secType := secVal.Type()
		for j := 0; j < secVal.NumField(); j++ {
			ff := secType.Field(j)
			fieldKey := getYAMLTagName(ff)
			dotKey := sectionKey + "." + fieldKey
			if excludedKeys[dotKey] {
				continue
			}
			if isSettableKind(secVal.Field(j).Kind()) {
				keys = append(keys, dotKey)
			}
		}
	}
	sort.Strings(keys)
	return keys
}

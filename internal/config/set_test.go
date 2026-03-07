package config

import (
	"strings"
	"testing"
)

func TestSetConfigKey_Success(t *testing.T) {
	cfg := DefaultUserPreferences()
	if cfg == nil {
		t.Fatal("DefaultUserPreferences returned nil")
	}

	tests := []struct {
		key    string
		value  string
		check  func(t *testing.T, c *AppConfig)
	}{
		{"theme.name", "catppuccin", func(t *testing.T, c *AppConfig) {
			if c.Theme.Name != "catppuccin" {
				t.Errorf("Theme.Name = %q, want catppuccin", c.Theme.Name)
			}
		}},
		{"Theme.Name", "arasaka", func(t *testing.T, c *AppConfig) {
			if c.Theme.Name != "arasaka" {
				t.Errorf("Theme.Name = %q, want arasaka", c.Theme.Name)
			}
		}},
		{"search.resultlimit", "50", func(t *testing.T, c *AppConfig) {
			if c.Search.ResultLimit != 50 {
				t.Errorf("Search.ResultLimit = %d, want 50", c.Search.ResultLimit)
			}
		}},
		{"search.enablepopularitysort", "false", func(t *testing.T, c *AppConfig) {
			if c.Search.EnablePopularitySort != false {
				t.Errorf("Search.EnablePopularitySort = %v, want false", c.Search.EnablePopularitySort)
			}
		}},
		{"configuration.defaulteditor", "code", func(t *testing.T, c *AppConfig) {
			if c.Configuration.DefaultEditor != "code" {
				t.Errorf("Configuration.DefaultEditor = %q, want code", c.Configuration.DefaultEditor)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.key+"="+tt.value, func(t *testing.T) {
			c := DefaultUserPreferences()
			if err := SetConfigKey(c, tt.key, tt.value); err != nil {
				t.Fatalf("SetConfigKey() error = %v", err)
			}
			tt.check(t, c)
		})
	}
}

func TestSetConfigKey_ExcludedKey(t *testing.T) {
	cfg := DefaultUserPreferences()
	excluded := []string{"version", "update.lastupdatecheck", "update.nextupdatecheck"}
	for _, key := range excluded {
		err := SetConfigKey(cfg, key, "any")
		if err == nil {
			t.Errorf("SetConfigKey(%q) expected error (excluded key)", key)
			continue
		}
		if !strings.Contains(err.Error(), "not settable") {
			t.Errorf("SetConfigKey(%q) error = %v, want message containing 'not settable'", key, err)
		}
	}
}

func TestSetConfigKey_UnknownKey(t *testing.T) {
	cfg := DefaultUserPreferences()
	err := SetConfigKey(cfg, "foo.bar", "x")
	if err == nil {
		t.Fatal("SetConfigKey(foo.bar) expected error")
	}
	if !strings.Contains(err.Error(), "unknown key") {
		t.Errorf("error = %v, want message containing 'unknown key'", err)
	}
}

func TestSetConfigKey_InvalidValue(t *testing.T) {
	cfg := DefaultUserPreferences()
	err := SetConfigKey(cfg, "theme.use256colorspace", "banana")
	if err == nil {
		t.Fatal("SetConfigKey(theme.use256colorspace=banana) expected error")
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("error = %v, want message containing 'invalid'", err)
	}
}

func TestSetConfigKey_InvalidKeyFormat(t *testing.T) {
	cfg := DefaultUserPreferences()
	err := SetConfigKey(cfg, "single", "x")
	if err == nil {
		t.Fatal("SetConfigKey(single) expected error")
	}
	if !strings.Contains(err.Error(), "section.field") {
		t.Errorf("error = %v, want message containing 'section.field'", err)
	}
}

func TestSettableKeys_ContainsExpected(t *testing.T) {
	keys := SettableKeys()
	wantContain := []string{"theme.name", "search.resultlimit", "configuration.defaulteditor"}
	for _, w := range wantContain {
		found := false
		for _, k := range keys {
			if k == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("SettableKeys() missing %q, got %v", w, keys)
		}
	}
}

func TestSettableKeys_ExcludesExcluded(t *testing.T) {
	keys := SettableKeys()
	excluded := []string{"version", "update.lastupdatecheck", "update.nextupdatecheck"}
	for _, ex := range excluded {
		for _, k := range keys {
			if k == ex {
				t.Errorf("SettableKeys() should not contain excluded key %q", ex)
			}
		}
	}
}

func TestSettableKeys_Sorted(t *testing.T) {
	keys := SettableKeys()
	for i := 1; i < len(keys); i++ {
		if keys[i] < keys[i-1] {
			t.Errorf("SettableKeys() not sorted: %q before %q", keys[i-1], keys[i])
		}
	}
}

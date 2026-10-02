package cmd

import (
	"strings"
	"testing"
)

func TestValidateSourceName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{name: "simple", input: "My Fonts", wantErr: ""},
		{name: "hyphen underscore", input: "My-Font_Source", wantErr: ""},
		{name: "empty", input: "   ", wantErr: "Name is required"},
		{name: "quoted empty", input: `""`, wantErr: "Name can only contain"},
		{name: "quotes around name", input: `"Roboto"`, wantErr: "Name can only contain"},
		{name: "path slash", input: "foo/bar", wantErr: "Name can only contain"},
		{name: "backslash", input: `foo\bar`, wantErr: "Name can only contain"},
		{name: "special chars", input: "Fonts!", wantErr: "Name can only contain"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSourceName(tt.input)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("ValidateSourceName(%q) = %v, want nil", tt.input, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("ValidateSourceName(%q) = nil, want error containing %q", tt.input, tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ValidateSourceName(%q) = %v, want containing %q", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateSourceForm_rejectsBadName(t *testing.T) {
	result := ValidateSourceForm(`""`, "https://example.com/fonts.json", "ex", nil, -1)
	if result.IsValid {
		t.Fatal("expected invalid form for quoted-empty name")
	}
	found := false
	for _, e := range result.Errors {
		if e.Field == "Name" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Name error, got %#v", result.Errors)
	}
}

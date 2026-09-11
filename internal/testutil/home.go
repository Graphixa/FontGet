package testutil

import "testing"

// SetHome sets HOME and USERPROFILE so os.UserHomeDir works on Unix and Windows.
func SetHome(t *testing.T, home string) {
	t.Helper()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
}

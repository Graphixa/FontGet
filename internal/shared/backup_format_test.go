package shared

import "testing"

func TestIsFontGetBackupComment(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want bool
	}{
		{BackupZipComment, true},
		{BackupZipComment + " ", true},
		{"  " + BackupZipComment, true},
		{BackupZipComment + ";extra", true},
		{"", false},
		{"   ", false},
		{"other archive", false},
		{"FontGet backup", false},
	}
	for _, tc := range cases {
		if got := IsFontGetBackupComment(tc.in); got != tc.want {
			t.Errorf("IsFontGetBackupComment(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

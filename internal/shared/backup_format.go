package shared

import "strings"

// BackupZipComment is written into FontGet backup zip archives as the global comment.
const BackupZipComment = "FontGet backup;format=1"

// IsFontGetBackupComment reports whether s is a FontGet backup zip comment.
func IsFontGetBackupComment(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	return s == BackupZipComment || strings.HasPrefix(s, BackupZipComment)
}

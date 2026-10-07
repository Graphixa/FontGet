package backupmeta

import "strings"

// ZipComment is written into FontGet backup zip archives as the global comment.
const ZipComment = "FontGet backup;format=1"

// IsBackupComment reports whether s is a FontGet backup zip comment.
func IsBackupComment(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	return s == ZipComment || strings.HasPrefix(s, ZipComment)
}

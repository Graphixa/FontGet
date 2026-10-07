package shared

import "fontget/internal/backupmeta"

// BackupZipComment is written into FontGet backup zip archives as the global comment.
const BackupZipComment = backupmeta.ZipComment

// IsFontGetBackupComment reports whether s is a FontGet backup zip comment.
func IsFontGetBackupComment(s string) bool {
	return backupmeta.IsBackupComment(s)
}

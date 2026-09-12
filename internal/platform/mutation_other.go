//go:build !windows

package platform

func undoFontRegistration(mut FileMutation) error {
	_ = mut
	return nil
}

func restorePriorFontRegistration(mut FileMutation) error {
	_ = mut
	return nil
}

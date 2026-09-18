//go:build windows

package platform

import "fmt"

func undoFontRegistration(mut FileMutation) error {
	if mut.RegistryAdded && mut.FontName != "" {
		fm, err := NewFontManager()
		if err == nil {
			if wm, ok := fm.(*windowsFontManager); ok {
				if rerr := wm.removeFontFromRegistry(mut.FontName); rerr != nil {
					return fmt.Errorf("remove registry entry %s: %w", mut.FontName, rerr)
				}
			}
		}
	}
	if mut.ResourceRegistered && mut.DestPath != "" {
		if err := RemoveFontResource(mut.DestPath); err != nil {
			return fmt.Errorf("unregister font resource %s: %w", mut.DestPath, err)
		}
	}
	return nil
}

func restorePriorFontRegistration(mut FileMutation) error {
	if !mut.PriorResourceRemoved || mut.DestPath == "" {
		return nil
	}
	if err := AddFontResource(mut.DestPath); err != nil {
		return fmt.Errorf("restore prior font resource %s: %w", mut.DestPath, err)
	}
	return nil
}

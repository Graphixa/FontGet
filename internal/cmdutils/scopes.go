package cmdutils

import (
	"fontget/internal/output"
	"fontget/internal/platform"
)

// DetectAccessibleScopes detects which font scopes are accessible based on elevation.
// Returns the list of scopes that can be accessed with the current privileges.
func DetectAccessibleScopes(fm platform.FontManager) ([]platform.InstallationScope, error) {
	isElevated, err := fm.IsElevated()
	if err != nil {
		// If we can't detect elevation, default to user scope only (safer)
		output.GetVerbose().Warning("Unable to detect elevation status: %v. Using user scope only.", err)
		return []platform.InstallationScope{platform.UserScope}, nil
	}

	if isElevated {
		// Admin/sudo - can access both scopes
		return []platform.InstallationScope{platform.UserScope, platform.MachineScope}, nil
	}

	// Regular user - only user scope
	return []platform.InstallationScope{platform.UserScope}, nil
}

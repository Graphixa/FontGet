package platform

// Logger is a minimal interface for logging.
// This allows platform to log without depending on the cmd package.
type Logger interface {
	Error(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Info(format string, args ...interface{})
}

// AutoDetectScope auto-detects installation scope based on elevation status.
// If elevation check fails, returns the defaultScope.
// If elevated, returns elevatedScope. Otherwise, returns "user".
//
// logger can be nil (for testing or when logging is not needed).
func AutoDetectScope(fontManager FontManager, defaultScope, elevatedScope string, logger Logger) (string, error) {
	isElevated, err := fontManager.IsElevated()
	if err != nil {
		if logger != nil {
			logger.Warn("Failed to detect elevation status: %v", err)
		}
		// Default to provided default scope if we can't detect elevation
		return defaultScope, nil
	}

	if isElevated {
		if logger != nil {
			logger.Info("Auto-detected elevated privileges, defaulting to '%s' scope", elevatedScope)
		}
		return elevatedScope, nil
	}

	if logger != nil {
		logger.Info("Auto-detected user privileges, defaulting to 'user' scope")
	}
	return "user", nil
}

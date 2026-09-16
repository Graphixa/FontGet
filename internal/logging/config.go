package logging

// DefaultConfig returns the default logging configuration
func DefaultConfig() Config {
	return Config{
		Level:      InfoLevel,
		MaxSize:    10,   // 10MB
		MaxBackups: 5,    // Keep 5 backup files
		MaxAge:     30,   // 30 days
		Compress:   true, // Compress old logs
	}
}

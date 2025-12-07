package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"
)

// LogLevel represents the severity level of a log message
type LogLevel int

const (
	// DebugLevel represents debug messages
	DebugLevel LogLevel = iota
	// InfoLevel represents informational messages
	InfoLevel
	// WarnLevel represents warning messages
	WarnLevel
	// ErrorLevel represents error messages
	ErrorLevel
)

var levelNames = map[LogLevel]string{
	DebugLevel: "DEBUG",
	InfoLevel:  "INFO",
	WarnLevel:  "WARN",
	ErrorLevel: "ERROR",
}

// Logger represents a logger instance
type Logger struct {
	mu            sync.Mutex
	level         LogLevel
	output        io.Writer
	file          *os.File
	maxSize       int64
	maxBackups    int
	maxAge        int
	compress      bool
	currentSize   int64
	lastRotation  time.Time
	rotationCount int
	ConsoleOutput bool
}

// Config holds the configuration for the logger
type Config struct {
	// Level is the minimum log level to record
	Level LogLevel
	// MaxSize is the maximum size in megabytes of the log file before it gets rotated
	MaxSize int
	// MaxBackups is the maximum number of old log files to retain
	MaxBackups int
	// MaxAge is the maximum number of days to retain old log files
	MaxAge int
	// Compress determines if the rotated log files should be compressed
	Compress bool
	// ConsoleOutput determines if debug/info logs should be printed to the console
	ConsoleOutput bool
}

var (
	globalLogger *Logger
	loggerOnce   sync.Once
)

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	loggerOnce.Do(func() {
		config := Config{
			Level:      InfoLevel,
			MaxSize:    10, // 10MB (restored to original)
			MaxBackups: 3,  // Reduced for testing cleanup
			MaxAge:     30, // 30 days
			Compress:   true,
		}
		var err error
		globalLogger, err = New(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
			// Create a fallback logger that writes to stderr
			globalLogger = &Logger{
				level:  config.Level,
				output: os.Stderr,
			}
		}
	})
	return globalLogger
}

// New creates a new logger instance using the default log directory
func New(config Config) (*Logger, error) {
	// Get the appropriate log directory based on OS
	logDir, err := getLogDirectory()
	if err != nil {
		return nil, fmt.Errorf("failed to get log directory: %w", err)
	}

	// Create the log file in default directory
	logFile := filepath.Join(logDir, "fontget.log")
	return NewWithPath(config, logFile)
}

// NewWithPath creates a new logger instance with a custom log file path
func NewWithPath(config Config, logFilePath string) (*Logger, error) {
	// Extract directory from log file path
	logDir := filepath.Dir(logFilePath)

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create the log file
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Get current file size
	fileInfo, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return &Logger{
		level:         config.Level,
		output:        file,
		file:          file,
		maxSize:       int64(config.MaxSize * 1024 * 1024), // Convert MB to bytes
		maxBackups:    config.MaxBackups,
		maxAge:        config.MaxAge,
		compress:      config.Compress,
		currentSize:   fileInfo.Size(),
		lastRotation:  time.Now(),
		ConsoleOutput: config.ConsoleOutput,
	}, nil
}

// Close closes the logger and its underlying file
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DebugLevel, format, args...)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(InfoLevel, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WarnLevel, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ErrorLevel, format, args...)
}

// log writes a log message with the given level
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if we need to rotate the log file
	if l.currentSize >= l.maxSize {
		if err := l.rotate(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to rotate log file: %v\n", err)
			return
		}
	}

	// Format the log message
	msg := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s: %s\n", timestamp, levelNames[level], msg)

	// Write the log entry
	if _, err := l.output.Write([]byte(logEntry)); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write log entry: %v\n", err)
		return
	}

	l.currentSize += int64(len(logEntry))

	// Print debug/info logs to console if ConsoleOutput is enabled
	if l.ConsoleOutput && (level == DebugLevel || level == InfoLevel) {
		fmt.Print(logEntry)
	}
}

// rotate rotates the log file
func (l *Logger) rotate() error {
	// Close the current file
	if err := l.file.Close(); err != nil {
		return fmt.Errorf("failed to close current log file: %w", err)
	}

	// Generate the new filename with timestamp
	timestamp := time.Now().Format("2006-01-02")
	dir := filepath.Dir(l.file.Name())
	base := filepath.Base(l.file.Name())
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]

	// If we've already rotated today, add a number
	if l.lastRotation.Format("2006-01-02") == timestamp {
		l.rotationCount++
	} else {
		l.rotationCount = 0
	}

	var newName string
	if l.rotationCount > 0 {
		newName = filepath.Join(dir, fmt.Sprintf("%s-%s.%d%s", name, timestamp, l.rotationCount, ext))
	} else {
		newName = filepath.Join(dir, fmt.Sprintf("%s-%s%s", name, timestamp, ext))
	}

	// Rename the current file
	if err := os.Rename(l.file.Name(), newName); err != nil {
		return fmt.Errorf("failed to rename log file: %w", err)
	}

	// Create a new file
	file, err := os.OpenFile(l.file.Name(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to create new log file: %w", err)
	}

	l.file = file
	l.output = file
	l.currentSize = 0
	l.lastRotation = time.Now()

	// Clean up old log files
	if err := l.cleanup(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to cleanup old log files: %v\n", err)
	}

	return nil
}

// cleanup removes old log files
func (l *Logger) cleanup() error {
	dir := filepath.Dir(l.file.Name())
	pattern := filepath.Join(dir, "fontget-*.log")

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to find log files: %w", err)
	}

	// Get current time for age calculations
	now := time.Now()
	var filesToRemove []string

	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			// Skip files we can't stat
			continue
		}

		// Check age constraint
		if l.maxAge > 0 {
			age := now.Sub(info.ModTime())
			if age > time.Duration(l.maxAge)*24*time.Hour {
				filesToRemove = append(filesToRemove, match)
				continue
			}
		}
	}

	// Sort remaining files by modification time (newest first) for backup count
	var remainingFiles []string
	for _, match := range matches {
		// Skip files already marked for removal due to age
		shouldSkip := false
		for _, toRemove := range filesToRemove {
			if match == toRemove {
				shouldSkip = true
				break
			}
		}
		if shouldSkip {
			continue
		}

		remainingFiles = append(remainingFiles, match)
	}

	// Sort by modification time (newest first)
	sort.Slice(remainingFiles, func(i, j int) bool {
		infoI, _ := os.Stat(remainingFiles[i])
		infoJ, _ := os.Stat(remainingFiles[j])
		return infoI.ModTime().After(infoJ.ModTime())
	})

	// Apply backup count constraint
	for i, match := range remainingFiles {
		if i >= l.maxBackups {
			filesToRemove = append(filesToRemove, match)
		}
	}

	// Remove files
	for _, file := range filesToRemove {
		if err := os.Remove(file); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to remove old log file %s: %v\n", file, err)
		}
	}

	return nil
}

// getLogDirectory returns the appropriate log directory for the current OS
func getLogDirectory() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "FontGet", "logs"), nil
	case "darwin":
		return filepath.Join(homeDir, "Library", "Logs", "fontget"), nil
	default: // Linux and others
		return filepath.Join(homeDir, ".local", "share", "fontget", "logs"), nil
	}
}

// GetLogDirectory returns the appropriate log directory for the current OS
func GetLogDirectory() (string, error) {
	return getLogDirectory()
}

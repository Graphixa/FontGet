package license

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"fontget/internal/config"
	"fontget/internal/repo"
)

// PromptForSourceAcceptance prompts the user to accept licenses for a source
func PromptForSourceAcceptance(sourceName string) (bool, error) {
	fmt.Println()
	fmt.Println("FontGet installs fonts from various sources. These fonts are subject to their respective license agreements.")
	fmt.Println()
	fmt.Printf("Do you accept the license agreements from the following sources:\n")
	fmt.Printf("- %s\n", sourceName)
	fmt.Println()
	fmt.Println("To review a particular font's license, run: fontget info <font-id> --license")
	fmt.Println()
	fmt.Print("Do you accept? (y/n): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes", nil
}

// CheckAndPromptForSource checks if a source is accepted and prompts if needed
func CheckAndPromptForSource(sourceName string) error {
	// Check if source is already accepted
	accepted, err := config.IsSourceAccepted(sourceName)
	if err != nil {
		return fmt.Errorf("failed to check source acceptance: %w", err)
	}

	if accepted {
		return nil // Source already accepted
	}

	// Prompt user for acceptance
	accepted, err = PromptForSourceAcceptance(sourceName)
	if err != nil {
		return fmt.Errorf("failed to prompt for license acceptance: %w", err)
	}

	if !accepted {
		return fmt.Errorf("license acceptance required to continue")
	}

	// Save acceptance
	if err := config.AcceptSource(sourceName); err != nil {
		return fmt.Errorf("failed to save license acceptance: %w", err)
	}

	return nil
}

// CheckFirstRunAndPrompt checks if this is the first run and prompts for Google Fonts acceptance
func CheckFirstRunAndPrompt() error {
	// Check if this is the first run
	isFirstRun, err := config.IsFirstRun()
	if err != nil {
		return fmt.Errorf("failed to check first run status: %w", err)
	}

	if !isFirstRun {
		return nil // Not first run, continue normally
	}

	// Show welcome message
	fmt.Println("Welcome to FontGet! This is your first time using the tool.")
	fmt.Println()

	// Check if Google Fonts is already accepted (shouldn't be on first run, but just in case)
	accepted, err := config.IsSourceAccepted("google-fonts")
	if err != nil {
		return fmt.Errorf("failed to check Google Fonts acceptance: %w", err)
	}

	if accepted {
		// Mark first run as completed and continue
		return config.MarkFirstRunCompleted()
	}

	// Prompt for Google Fonts acceptance
	if err := CheckAndPromptForSource("google-fonts"); err != nil {
		return err
	}

	// Mark first run as completed
	return config.MarkFirstRunCompleted()
}

// GetLicenseURL returns the license URL for a font from a specific source
func GetLicenseURL(fontName, source string) string {
	switch source {
	case "google-fonts":
		// Google Fonts OFL license pattern
		normalizedName := strings.ToLower(strings.ReplaceAll(fontName, " ", ""))
		return fmt.Sprintf("https://raw.githubusercontent.com/google/fonts/main/ofl/%s/OFL.txt", normalizedName)
	// Future sources can be added here
	default:
		return ""
	}
}

// FetchLicenseText fetches license text from a URL with cross-platform compatibility
func FetchLicenseText(url string) (string, error) {
	return repo.FetchURLContent(url)
}

// DisplayLicenseText displays license text with cross-platform pagination
func DisplayLicenseText(content string) error {
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		fmt.Println(line)

		// Pagination every 20 lines (cross-platform)
		if (i+1)%20 == 0 && i < len(lines)-1 {
			fmt.Print("\nPress Enter to continue...")
			reader := bufio.NewReader(os.Stdin)
			reader.ReadString('\n') // Wait for Enter key
		}
	}

	return nil
}

// HandleLicenseError displays a user-friendly error message for license issues
func HandleLicenseError(fontName string, err error) {
	fmt.Printf("License not found for \"%s\". Please review the license agreement for this font before using it.\n\n", fontName)
	fmt.Println("You can try:")
	fmt.Printf("- fontget info \"%s\" (to see available license info)\n", fontName)
	fmt.Println("- Visit the font's source URL for license details")
}

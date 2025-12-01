package license

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"fontget/internal/components"
	"fontget/internal/config"
	"fontget/internal/repo"
	"fontget/internal/ui"
)

// PromptForSourceAcceptance prompts the user to accept licenses for a source
// This function uses styled UI components for better presentation
func PromptForSourceAcceptance(sourceName string) (bool, error) {
	// Display styled license information
	fmt.Println()
	fmt.Println(ui.PageTitle.Render("License Agreement"))
	fmt.Println()
	fmt.Println(ui.Text.Render("FontGet installs fonts from various sources."))
	fmt.Println(ui.Text.Render("These fonts are subject to their respective license agreements."))
	fmt.Println()
	fmt.Printf("%s %s\n", ui.InfoText.Render("Source:"), ui.TableSourceName.Render(sourceName))
	fmt.Println()
	fmt.Println(ui.Text.Render("To review a particular font's license, run:"))
	fmt.Printf("  %s\n", ui.CommandExample.Render("fontget info <font-id> --license"))
	fmt.Println()

	// Use confirmation dialog for better UX
	message := fmt.Sprintf("Do you accept the license agreements from %s?", ui.TableSourceName.Render(sourceName))
	confirmed, err := components.RunConfirm(
		"",
		message,
	)
	if err != nil {
		// Fallback to basic prompt if confirmation dialog fails
		// User-friendly error handling per verbose/debug guidelines
		return promptForSourceAcceptanceFallback(sourceName)
	}

	// Section ends - confirmation dialog handles its own spacing via alt screen
	return confirmed, nil
}

// promptForSourceAcceptanceFallback provides a basic fallback if the confirmation dialog fails
func promptForSourceAcceptanceFallback(sourceName string) (bool, error) {
	fmt.Print("Do you accept? (y/n): ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		// User-friendly error message per verbose/debug guidelines
		return false, fmt.Errorf("unable to read response: %w", err)
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
		// User-friendly error message per verbose/debug guidelines
		return fmt.Errorf("unable to prompt for license acceptance: %w", err)
	}

	if !accepted {
		return fmt.Errorf("license acceptance required to continue")
	}

	// Save acceptance
	if err := config.AcceptSource(sourceName); err != nil {
		// User-friendly error message per verbose/debug guidelines
		return fmt.Errorf("unable to save license acceptance: %w", err)
	}

	return nil
}

// CheckFirstRunAndPrompt checks if this is the first run and prompts for Google Fonts acceptance
// DEPRECATED: This function is kept for backward compatibility.
// New code should use onboarding.RunFirstRunOnboarding() directly from cmd/root.go
func CheckFirstRunAndPrompt() error {
	// Check if this is the first run
	isFirstRun, err := config.IsFirstRun()
	if err != nil {
		return fmt.Errorf("failed to check first run status: %w", err)
	}

	if !isFirstRun {
		return nil // Not first run, continue normally
	}

	// Show welcome message with styled UI
	fmt.Println()
	fmt.Println(ui.PageTitle.Render("Welcome to FontGet!"))
	fmt.Println()
	fmt.Println(ui.Text.Render("This is your first time using FontGet. Let's get you set up."))
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

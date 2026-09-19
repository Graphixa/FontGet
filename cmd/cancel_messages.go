package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"fontget/internal/shared"
	"fontget/internal/ui"
)

// Cancellation contract user-facing copy (single source — do not duplicate in platform files).
const (
	msgInstallationCancelledIncomplete = "Installation cancelled... Some fonts were not installed."
	msgRemovalCancelledIncomplete      = "Removal cancelled... Some fonts were not removed."
	msgInstallationCancelledShort      = "Installation cancelled"
	msgRemovalCancelledShort           = "Removal cancelled"
	msgDownloadCancelledShort          = "Download cancelled"
	msgForceRemovalCancelledShort      = "Force removal cancelled"
)

// IsCancelErr reports user cancellation (context or FontGet sentinel).
func IsCancelErr(err error) bool {
	return err != nil && (errors.Is(err, shared.ErrOperationCancelled) || errors.Is(err, context.Canceled) || errors.Is(err, ui.ErrCancelled))
}

func shellQuoteArg(arg string) string {
	if arg == "" {
		return `""`
	}
	needs := false
	for _, r := range arg {
		if unicode.IsSpace(r) || strings.ContainsRune(`"'&|<>()^%!`, r) {
			needs = true
			break
		}
	}
	if !needs {
		return arg
	}
	return `"` + strings.ReplaceAll(arg, `"`, `\"`) + `"`
}

func dedupePackageIDs(ids []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		key := strings.ToLower(id)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, id)
	}
	return out
}

func formatRetryCommand(verb string, packageIDs []string, scope string, force bool) string {
	parts := []string{"fontget", verb}
	for _, id := range dedupePackageIDs(packageIDs) {
		parts = append(parts, shellQuoteArg(id))
	}
	scope = strings.TrimSpace(scope)
	if scope != "" && !strings.EqualFold(scope, "user") {
		parts = append(parts, "--scope", shellQuoteArg(scope))
	}
	if force {
		parts = append(parts, "--force")
	}
	return strings.Join(parts, " ")
}

// FormatRetryAddCommand builds the suggested fontget add command for incomplete installs.
func FormatRetryAddCommand(packageIDs []string, scope string, force bool) string {
	return formatRetryCommand("add", packageIDs, scope, force)
}

// FormatRetryRemoveCommand builds the suggested fontget remove command for incomplete removals.
func FormatRetryRemoveCommand(packageIDs []string, scope string) string {
	return formatRetryCommand("remove", packageIDs, scope, false)
}

// FormatInstallationCancelledText returns the full cancellation + retry text for installs / force installs.
func FormatInstallationCancelledText(packageIDs []string, scope string, force bool) string {
	ids := dedupePackageIDs(packageIDs)
	if len(ids) == 0 {
		return msgInstallationCancelledShort
	}
	return fmt.Sprintf("%s\nRun `%s` again to complete the installation.",
		msgInstallationCancelledIncomplete, FormatRetryAddCommand(ids, scope, force))
}

// FormatRemovalCancelledText returns the full cancellation + retry text for removals.
func FormatRemovalCancelledText(packageIDs []string, scope string) string {
	ids := dedupePackageIDs(packageIDs)
	if len(ids) == 0 {
		return msgRemovalCancelledShort
	}
	return fmt.Sprintf("%s\nRun `%s` again to complete the removal.",
		msgRemovalCancelledIncomplete, FormatRetryRemoveCommand(ids, scope))
}

func printCancelledText(text string) {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return
	}
	fmt.Printf("%s\n", ui.WarningText.Render(lines[0]))
	for _, line := range lines[1:] {
		fmt.Println(line)
	}
	fmt.Println()
}

// PrintInstallationCancelledMessage prints the contract cancellation wording for add / add --force / import.
func PrintInstallationCancelledMessage(packageIDs []string, scope string, force bool) {
	printCancelledText(FormatInstallationCancelledText(packageIDs, scope, force))
}

// PrintRemovalCancelledMessage prints the contract cancellation wording for remove.
func PrintRemovalCancelledMessage(packageIDs []string, scope string) {
	printCancelledText(FormatRemovalCancelledText(packageIDs, scope))
}

// FinishInstallationCancel handles exit status after an install-family cancel.
// Incomplete work → print contract message and return non-zero (AlreadyPrinted).
// No remaining work → return nil so the command reports completion.
func FinishInstallationCancel(incompletePackageIDs []string, scope string, force bool) error {
	ids := dedupePackageIDs(incompletePackageIDs)
	if len(ids) == 0 {
		return nil
	}
	PrintInstallationCancelledMessage(ids, scope, force)
	return shared.AlreadyPrinted(shared.ErrOperationCancelled)
}

// FinishRemovalCancel handles exit status after a remove cancel.
func FinishRemovalCancel(incompletePackageIDs []string, scope string) error {
	ids := dedupePackageIDs(incompletePackageIDs)
	if len(ids) == 0 {
		return nil
	}
	PrintRemovalCancelledMessage(ids, scope)
	return shared.AlreadyPrinted(shared.ErrOperationCancelled)
}

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

// dedupeExactStrings preserves order and case; identical strings collapse once.
func dedupeExactStrings(ids []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func formatRetryCommand(verb string, args []string, scope string, force bool) string {
	parts := []string{"fontget", verb}
	for _, id := range args {
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
// Catalog Font IDs are deduped case-insensitively.
func FormatRetryAddCommand(packageIDs []string, scope string, force bool) string {
	return formatRetryCommand("add", dedupePackageIDs(packageIDs), scope, force)
}

// FormatRetryAddCommandMixed builds a retry command with catalog IDs (case-insensitive)
// and local paths (exact case preserved).
func FormatRetryAddCommandMixed(catalogIDs, localPaths []string, scope string, force bool) string {
	args := append(dedupePackageIDs(catalogIDs), dedupeExactStrings(localPaths)...)
	return formatRetryCommand("add", args, scope, force)
}

// FormatRetryRemoveCommand builds the suggested fontget remove command for incomplete removals.
func FormatRetryRemoveCommand(packageIDs []string, scope string) string {
	return formatRetryCommand("remove", dedupePackageIDs(packageIDs), scope, false)
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

// FormatInstallationCancelledTextMixed formats cancel guidance for mixed catalog + local retries.
func FormatInstallationCancelledTextMixed(catalogIDs, localPaths, missingLocal []string, scope string, force bool) string {
	catalogIDs = dedupePackageIDs(catalogIDs)
	localPaths = dedupeExactStrings(localPaths)
	missingLocal = dedupeExactStrings(missingLocal)
	if len(catalogIDs) == 0 && len(localPaths) == 0 {
		msg := msgInstallationCancelledIncomplete
		if len(missingLocal) > 0 {
			msg += "\nProvide the original local font files and run fontget add again."
		} else {
			return msgInstallationCancelledShort
		}
		return msg
	}
	cmd := FormatRetryAddCommandMixed(catalogIDs, localPaths, scope, force)
	text := fmt.Sprintf("%s\nRun `%s` again to complete the installation.", msgInstallationCancelledIncomplete, cmd)
	if len(missingLocal) > 0 {
		text += "\nSome original local inputs are no longer available and must be supplied again."
	}
	return text
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

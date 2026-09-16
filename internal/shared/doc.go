// Package shared contains general-purpose helpers shared across commands and internal packages.
//
// Keep this package CLI-agnostic: no Cobra assumptions, no prompting, and no direct terminal UI.
// For feature/domain-specific helpers (e.g. sources management), keep helpers in the owning package.
package shared

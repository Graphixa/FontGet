package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestFishCompletionGeneration(t *testing.T) {
	var buf bytes.Buffer
	if err := rootCmd.GenFishCompletion(&buf, true); err != nil {
		t.Fatalf("GenFishCompletion: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "fish completion for fontget") {
		t.Fatalf("expected fish completion header in output, got: %s", truncateCompletionTestOutput(out, 400))
	}
}

func TestSkipShellCompCommandsDocumented(t *testing.T) {
	// If these drift, PersistentPreRun must still skip shell completion invocations.
	if cobra.ShellCompRequestCmd != "__complete" {
		t.Fatalf("unexpected ShellCompRequestCmd: %q", cobra.ShellCompRequestCmd)
	}
	if cobra.ShellCompNoDescRequestCmd != "__completeNoDesc" {
		t.Fatalf("unexpected ShellCompNoDescRequestCmd: %q", cobra.ShellCompNoDescRequestCmd)
	}
}

func truncateCompletionTestOutput(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

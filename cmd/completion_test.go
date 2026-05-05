package cmd

import (
	"bytes"
	"os"
	"path/filepath"
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

func TestPrependZshFpathBlock_OrderAndMarker(t *testing.T) {
	got := prependZshFpathBlock(`# user config
compinit
`)
	if !strings.HasPrefix(got, zshFpathMarker) {
		t.Fatalf("expected zsh fpath block first, got:\n%s", got)
	}
	if !strings.Contains(got, `fpath=("${HOME}/.zsh/completions" $fpath)`) {
		t.Fatalf("expected HOME-based fpath line, got:\n%s", got)
	}
	if !strings.Contains(got, "compinit") {
		t.Fatalf("expected original .zshrc preserved after block, got:\n%s", got)
	}
}

func TestUpdateShellConfig_ZshPrependsMarker(t *testing.T) {
	dir := t.TempDir()
	zshrc := filepath.Join(dir, ".zshrc")
	if err := os.WriteFile(zshrc, []byte("source $ZSH/oh-my-zsh.sh\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := updateShellConfig(zshrc, "", "zsh"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(zshrc)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.HasPrefix(s, zshFpathMarker) {
		t.Fatalf("expected marker at start of .zshrc, got:\n%s", s)
	}
	if err := updateShellConfig(zshrc, "", "zsh"); err == nil || !strings.Contains(err.Error(), "already installed") {
		t.Fatalf("expected already installed error on second run, got err=%v", err)
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

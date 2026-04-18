package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestCardInnerContentWidth(t *testing.T) {
	if w := CardInnerContentWidth(80, 2); w != 76 {
		t.Fatalf("got %d want 76", w)
	}
	if w := CardInnerContentWidth(4, 2); w != 1 {
		t.Fatalf("got %d want 1", w)
	}
}

func TestChunkURLForWidth_prefersSlashBreaks(t *testing.T) {
	u := "https://example.com/foo/bar/baz"
	chunks := chunkURLForWidth(u, 24, 24)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	if strings.Join(chunks, "") != u {
		t.Fatalf("rejoin mismatch: %q", strings.Join(chunks, ""))
	}
	for _, c := range chunks {
		if ansi.StringWidth(c) > 24 {
			t.Fatalf("chunk too wide: %q width %d", c, ansi.StringWidth(c))
		}
	}
}

func TestBuildCardSectionContent_uniformWidth(t *testing.T) {
	inner := 40
	body := buildCardSectionContent([]CardSection{
		{Label: "Name", Value: "X"},
		{Label: "Source URL", Value: "https://example.com/a/b", IsURL: true},
	}, inner)
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		if got := ansi.StringWidth(line); got != inner {
			t.Fatalf("line width %d != inner %d: %q", got, inner, line)
		}
	}
}

func TestBreakURLLine_hardBreaksWhenNoSlash(t *testing.T) {
	s := "abcdefghij"
	idx := breakURLLine(s, 3)
	if idx < 1 {
		t.Fatalf("expected positive cut, got %d", idx)
	}
	if ansi.StringWidth(s[:idx]) > 3 {
		t.Fatalf("width %d > 3", ansi.StringWidth(s[:idx]))
	}
}

package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestListInstalledFonts_FlatAndNested(t *testing.T) {
	root := t.TempDir()
	flat := filepath.Join(root, "Roboto-Regular.ttf")
	nestedDir := filepath.Join(root, "nested")
	if err := os.Mkdir(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(nestedDir, "JetBrainsMono-Regular.otf")
	ignored := filepath.Join(root, "readme.txt")
	for _, p := range []string{flat, nested, ignored} {
		if err := os.WriteFile(p, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := ListInstalledFonts(root)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, name := range got {
		found[name] = true
	}
	if !found["Roboto-Regular.ttf"] {
		t.Fatalf("missing flat font, got %v", got)
	}
	nestedRel := filepath.Join("nested", "JetBrainsMono-Regular.otf")
	if !found[nestedRel] {
		t.Fatalf("missing nested font %q, got %v", nestedRel, got)
	}
	if found["readme.txt"] {
		t.Fatal("non-font file should be ignored")
	}
}

func TestListInstalledFonts_WindowsUserAndSystemDirs(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows font directories")
	}
	fm, err := NewFontManager()
	if err != nil {
		t.Fatal(err)
	}
	for _, scope := range []InstallationScope{UserScope, MachineScope} {
		dir := fm.GetFontDir(scope)
		names, err := ListInstalledFonts(dir)
		if err != nil {
			t.Fatalf("ListInstalledFonts %s (%s): %v", scope, dir, err)
		}
		if len(names) == 0 {
			t.Fatalf("%s font dir %s: expected installed fonts", scope, dir)
		}
	}
}

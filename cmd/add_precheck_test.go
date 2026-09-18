package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"fontget/internal/installations"
	"fontget/internal/platform"
	"fontget/internal/testutil"
)

func TestCheckInstalledViaRegistry_allFilesPresent(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	file := filepath.Join(fontDir, "Noto.ttf")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := installations.RecordInstallation(installations.RecordParams{
		FontID:      "nerd.noto",
		CatalogName: "Noto",
		Scope:       "user",
		Files: []installations.InstalledFontFile{
			{Path: file, SFNT: installations.SFNTSnapshot{Family: "Noto"}},
		},
	}); err != nil {
		t.Fatal(err)
	}

	ok, handled := checkInstalledViaRegistry("nerd.noto", platform.UserScope)
	if !handled || !ok {
		t.Fatalf("got installed=%v handled=%v want true,true", ok, handled)
	}
	// Case-insensitive Font ID
	ok, handled = checkInstalledViaRegistry("NERD.NOTO", platform.UserScope)
	if !handled || !ok {
		t.Fatalf("case-insensitive: installed=%v handled=%v", ok, handled)
	}
}

func TestCheckInstalledViaRegistry_missingFile(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	missing := filepath.Join(t.TempDir(), "gone.ttf")
	if err := installations.RecordInstallation(installations.RecordParams{
		FontID: "nerd.noto",
		Scope:  "user",
		Files: []installations.InstalledFontFile{
			{Path: missing, SFNT: installations.SFNTSnapshot{Family: "Noto"}},
		},
	}); err != nil {
		t.Fatal(err)
	}

	ok, handled := checkInstalledViaRegistry("nerd.noto", platform.UserScope)
	if !handled || ok {
		t.Fatalf("got installed=%v handled=%v want false,true", ok, handled)
	}
}

func TestCheckInstalledViaRegistry_wrongScope(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	file := filepath.Join(fontDir, "Noto.ttf")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := installations.RecordInstallation(installations.RecordParams{
		FontID: "nerd.noto",
		Scope:  "user",
		Files: []installations.InstalledFontFile{
			{Path: file, SFNT: installations.SFNTSnapshot{Family: "Noto"}},
		},
	}); err != nil {
		t.Fatal(err)
	}

	ok, handled := checkInstalledViaRegistry("nerd.noto", platform.MachineScope)
	if !handled || ok {
		t.Fatalf("got installed=%v handled=%v want false,true", ok, handled)
	}
}

func TestCheckInstalledViaRegistry_noRecord(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)

	ok, handled := checkInstalledViaRegistry("nerd.missing", platform.UserScope)
	if handled || ok {
		t.Fatalf("got installed=%v handled=%v want false,false", ok, handled)
	}
}

func TestCheckFontsAlreadyInstalled_registryShortCircuit(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	file := filepath.Join(fontDir, "Pack.ttf")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := installations.RecordInstallation(installations.RecordParams{
		FontID: "nerd.pack",
		Scope:  "user",
		Files: []installations.InstalledFontFile{
			{Path: file, SFNT: installations.SFNTSnapshot{Family: "Pack"}},
		},
	}); err != nil {
		t.Fatal(err)
	}

	// nil fontManager is fine: registry hit must not scan.
	ok, err := checkFontsAlreadyInstalled("nerd.pack", "Pack", platform.UserScope, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected already installed via registry")
	}
}

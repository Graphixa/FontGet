//go:build windows

package platform

import (
	"bytes"
	"path/filepath"
	"syscall"
	"testing"
)

func TestFontRegistryRoot(t *testing.T) {
	for _, tc := range []struct {
		scope InstallationScope
		want  uintptr
	}{{UserScope, 0x80000001}, {MachineScope, HKEY_LOCAL_MACHINE}} {
		got, err := fontRegistryRoot(tc.scope)
		if err != nil || got != tc.want {
			t.Fatalf("scope %s: got %x, %v; want %x", tc.scope, got, err, tc.want)
		}
	}
	if _, err := fontRegistryRoot(InstallationScope("invalid")); err == nil {
		t.Fatal("expected invalid scope error")
	}
}

func TestUserFontRegistryLifecycle(t *testing.T) {
	m := &windowsFontManager{}
	path := filepath.Join(t.TempDir(), "FontGet-test.ttf")
	name := "FontGet-test-" + filepath.Base(filepath.Dir(filepath.Dir(path)))
	prior, err := m.captureFontRegistryValue(name, UserScope)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if prior.Found {
			_ = m.restoreFontRegistryValue(prior, UserScope)
		} else {
			_ = m.removeFontFromRegistry(name, UserScope)
		}
	})
	if err := m.addFontToRegistry(name, path, UserScope); err != nil {
		t.Fatal(err)
	}
	value, err := m.captureFontRegistryValue(name, UserScope)
	if err != nil || !value.Found {
		t.Fatalf("capture: %+v, %v", value, err)
	}
	encoded, err := syscall.UTF16FromString(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(value.Raw, uint16SliceAsBytes(encoded)) || value.Type != REG_SZ {
		t.Fatal("registry value must contain the full UTF-16 path")
	}
	if err := m.removeFontFromRegistry(name, UserScope); err != nil {
		t.Fatal(err)
	}
	if absent, err := m.captureFontRegistryValue(name, UserScope); err != nil || absent.Found {
		t.Fatalf("expected absent entry: %+v, %v", absent, err)
	}
	if err := m.restoreFontRegistryValue(value, UserScope); err != nil {
		t.Fatal(err)
	}
	if restored, err := m.captureFontRegistryValue(name, UserScope); err != nil || !bytes.Equal(restored.Raw, value.Raw) {
		t.Fatalf("restore: %+v, %v", restored, err)
	}
}

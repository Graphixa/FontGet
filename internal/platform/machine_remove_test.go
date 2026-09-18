//go:build windows

package platform

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"testing"
)

func TestRegSZByteLenUsesUTF16NotUTF8(t *testing.T) {
	s := "カフェ.ttf" // multibyte UTF-8; UTF-16 length differs from (len(s)+1)*2
	got, err := regSZByteLen(s)
	if err != nil {
		t.Fatal(err)
	}
	u16, err := syscall.UTF16FromString(s)
	if err != nil {
		t.Fatal(err)
	}
	want := len(u16) * 2
	if got != want {
		t.Fatalf("got %d want %d", got, want)
	}
	oldBug := (len(s) + 1) * 2
	if got == oldBug {
		t.Fatalf("UTF-8-based length %d must not equal UTF-16 length for %q", oldBug, s)
	}
}

func TestUint16SliceAsBytesMatchesUTF16FromString(t *testing.T) {
	s := "café/Fonts/顔.ttf"
	u16, err := syscall.UTF16FromString(s)
	if err != nil {
		t.Fatal(err)
	}
	raw := uint16SliceAsBytes(u16)
	if len(raw) != len(u16)*2 {
		t.Fatalf("len raw=%d u16=%d", len(raw), len(u16))
	}
	// Round-trip first code unit.
	if raw[0] != byte(u16[0]) || raw[1] != byte(u16[0]>>8) {
		t.Fatalf("endian packing wrong: %v vs %x", raw[:2], u16[0])
	}
}

func TestRemoveMachineScopedFont_DeleteFailRestoresExactRegistryBytes(t *testing.T) {
	u16, err := syscall.UTF16FromString("カフェ.ttf")
	if err != nil {
		t.Fatal(err)
	}
	priorRaw := uint16SliceAsBytes(u16)
	prior := registryFontValue{Name: "Face.ttf (TrueType)", Raw: priorRaw, Type: REG_SZ, Found: true}
	var restored registryFontValue
	err = removeMachineScopedFont("Face.ttf", `C:\Fonts\Face.ttf`, machineRemoveOps{
		RemoveGDI:  func(string) error { return nil },
		AddGDI:     func(string) error { return nil },
		CaptureReg: func(string) (registryFontValue, error) { return prior, nil },
		DeleteReg:  func(string) error { return nil },
		RestoreReg: func(v registryFontValue) error {
			restored = v
			return nil
		},
		RemoveFile:   func(string) error { return errors.New("access denied") },
		NotifyChange: func() error { return nil },
	})
	if err == nil {
		t.Fatal("expected delete failure")
	}
	if !bytes.Equal(restored.Raw, priorRaw) || restored.Type != REG_SZ || restored.Name != prior.Name {
		t.Fatalf("restore must use exact captured bytes: %+v", restored)
	}
}

func TestRemoveMachineScopedFont_RegistryDeleteFailStopsAndRestoresGDI(t *testing.T) {
	var gdiAdded, deletedFile bool
	err := removeMachineScopedFont("Face.ttf", `C:\Fonts\Face.ttf`, machineRemoveOps{
		RemoveGDI: func(string) error { return nil },
		AddGDI: func(string) error {
			gdiAdded = true
			return nil
		},
		CaptureReg: func(string) (registryFontValue, error) {
			return registryFontValue{Name: "n", Raw: []byte{1, 0}, Type: REG_SZ, Found: true}, nil
		},
		DeleteReg: func(string) error { return errors.New("access denied") },
		RestoreReg: func(registryFontValue) error {
			t.Fatal("should not restore registry when delete never succeeded")
			return nil
		},
		RemoveFile: func(string) error {
			deletedFile = true
			return nil
		},
		NotifyChange: func() error { return nil },
	})
	if err == nil {
		t.Fatal("expected registry failure")
	}
	if deletedFile {
		t.Fatal("must not delete file after registry removal failure")
	}
	if !gdiAdded {
		t.Fatal("must restore GDI")
	}
}

func TestRemoveMachineScopedFont_DeleteFailSurfacesRepairError(t *testing.T) {
	err := removeMachineScopedFont("Face.ttf", `C:\Fonts\Face.ttf`, machineRemoveOps{
		RemoveGDI: func(string) error { return nil },
		AddGDI:    func(string) error { return errors.New("gdi restore boom") },
		CaptureReg: func(string) (registryFontValue, error) {
			return registryFontValue{Name: "n", Raw: []byte{1, 0}, Type: REG_SZ, Found: true}, nil
		},
		DeleteReg:    func(string) error { return nil },
		RestoreReg:   func(registryFontValue) error { return errors.New("reg restore boom") },
		RemoveFile:   func(string) error { return errors.New("delete boom") },
		NotifyChange: func() error { return nil },
	})
	if err == nil {
		t.Fatal("expected combined error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "delete boom") || !strings.Contains(msg, "restore") {
		t.Fatalf("expected primary+repair in err, got %v", err)
	}
}

func TestRemoveMachineScopedFont_AbsentRegistryIsIdempotent(t *testing.T) {
	err := removeMachineScopedFont("Face.ttf", `C:\Fonts\Face.ttf`, machineRemoveOps{
		RemoveGDI: func(string) error { return nil },
		AddGDI:    func(string) error { return fmt.Errorf("should not restore") },
		CaptureReg: func(string) (registryFontValue, error) {
			return registryFontValue{Name: "Face.ttf (TrueType)", Found: false}, nil
		},
		DeleteReg: func(string) error {
			t.Fatal("delete should be skipped when not found")
			return nil
		},
		RestoreReg: func(registryFontValue) error {
			t.Fatal("restore should be skipped when not found")
			return nil
		},
		RemoveFile:   func(string) error { return os.ErrNotExist },
		NotifyChange: func() error { return nil },
	})
	if err != nil {
		t.Fatalf("absent registry + absent file should succeed: %v", err)
	}
}

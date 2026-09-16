//go:build windows

package platform

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

// ErrRegistryValueAbsent means the Fonts registry value was already gone (idempotent OK).
var ErrRegistryValueAbsent = errors.New("registry font value absent")

// registryFontValue is a captured machine-scope Fonts registry entry.
type registryFontValue struct {
	Name  string
	Data  string
	Type  uint32
	Found bool
}

// machineRemoveOps are narrow seams for machine-scope removal (tests inject fakes).
type machineRemoveOps struct {
	RemoveGDI      func(path string) error
	AddGDI         func(path string) error
	CaptureReg     func(fontName string) (registryFontValue, error)
	DeleteReg      func(fontName string) error
	RestoreReg     func(v registryFontValue) error
	RemoveFile     func(path string) error
	NotifyChange   func() error
	UnregisterOnly bool
	SkipNotify     bool
}

func isBenignGDIRemoveErr(err error) bool {
	if err == nil {
		return true
	}
	s := err.Error()
	return strings.Contains(s, "error code: 0") || strings.Contains(s, "The operation completed successfully")
}

func joinRemoveRepairErr(primary error, repair error) error {
	if primary == nil {
		return repair
	}
	if repair == nil {
		return primary
	}
	return fmt.Errorf("%w (also failed to restore registration: %v)", primary, repair)
}

// removeMachineScopedFont removes GDI + registry then the file, restoring prior
// registration when deletion fails. Registry removal failure stops before delete.
func removeMachineScopedFont(fontName, fontPath string, ops machineRemoveOps) error {
	gdiRemoved := false
	if err := ops.RemoveGDI(fontPath); err != nil {
		if !isBenignGDIRemoveErr(err) {
			return fmt.Errorf("failed to remove font resource: %w", err)
		}
	} else {
		gdiRemoved = true
	}

	prior, capErr := ops.CaptureReg(fontName)
	if capErr != nil {
		var repair error
		if gdiRemoved {
			repair = ops.AddGDI(fontPath)
		}
		return joinRemoveRepairErr(fmt.Errorf("failed to capture font registry state: %w", capErr), repair)
	}

	if prior.Found {
		if err := ops.DeleteReg(fontName); err != nil && !errors.Is(err, ErrRegistryValueAbsent) {
			var repair error
			if gdiRemoved {
				repair = ops.AddGDI(fontPath)
			}
			return joinRemoveRepairErr(fmt.Errorf("failed to remove font from registry: %w", err), repair)
		}
	}

	if ops.UnregisterOnly {
		return nil
	}

	if ops.NotifyChange != nil {
		_ = ops.NotifyChange()
	}

	var removeErr error
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*40) * time.Millisecond)
			if ops.NotifyChange != nil {
				_ = ops.NotifyChange()
			}
		}
		removeErr = ops.RemoveFile(fontPath)
		if removeErr == nil || os.IsNotExist(removeErr) {
			removeErr = nil
			break
		}
	}
	if removeErr != nil {
		var repair error
		if prior.Found {
			if rerr := ops.RestoreReg(prior); rerr != nil {
				repair = rerr
			}
		}
		if gdiRemoved {
			if aerr := ops.AddGDI(fontPath); aerr != nil {
				repair = joinRemoveRepairErr(repair, aerr)
			}
		}
		return joinRemoveRepairErr(fmt.Errorf("failed to remove font file: %w", removeErr), repair)
	}

	if !ops.SkipNotify && ops.NotifyChange != nil {
		if err := ops.NotifyChange(); err != nil {
			return fmt.Errorf("failed to notify font change: %w", err)
		}
	}
	return nil
}

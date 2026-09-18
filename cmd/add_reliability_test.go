package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"fontget/internal/installations"
	"fontget/internal/platform"
	"fontget/internal/repo"
	"fontget/internal/shared"
	"fontget/internal/testutil"
)

func TestAddMissingFontIDExitNonZero(t *testing.T) {
	rootCmd.SetArgs([]string{"add"})
	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	err := rootCmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "font ID is required") {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestAddHelpExitZeroSubprocess(t *testing.T) {
	bin := buildTestFontget(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "add", "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("help failed: %v\n%s", err, out)
	}
	if cmd.ProcessState.ExitCode() != 0 {
		t.Fatalf("help exit %d", cmd.ProcessState.ExitCode())
	}
}

func TestAddNoArgsSubprocessNonZero(t *testing.T) {
	bin := buildTestFontget(t)
	home := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "add")
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"USERPROFILE="+home,
		"FONTGET_ACCEPT_AGREEMENTS=1",
		"FONTGET_ACCEPT_DEFAULTS=1",
	)
	cmd.Stdin = bytes.NewReader(nil)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit")
	}
	if cmd.ProcessState.ExitCode() == 0 {
		t.Fatal("exit 0")
	}
	combined := stdout.String() + stderr.String()
	if strings.Contains(combined, "?1049") {
		t.Fatalf("interactive escape sequences in piped output: %q", combined)
	}
}

func TestAddDebugMissingIDNonZero(t *testing.T) {
	bin := buildTestFontget(t)
	home := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "add", "--debug")
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"USERPROFILE="+home,
		"FONTGET_ACCEPT_AGREEMENTS=1",
		"FONTGET_ACCEPT_DEFAULTS=1",
	)
	cmd.Stdin = bytes.NewReader(nil)
	if err := cmd.Run(); err == nil {
		t.Fatal("expected non-zero")
	}
}

var (
	testBinOnce sync.Once
	testBinPath string
	testBinErr  error
)

func buildTestFontget(t *testing.T) string {
	t.Helper()
	testBinOnce.Do(func() {
		dir, err := os.MkdirTemp("", "fontget-reltest-*")
		if err != nil {
			testBinErr = err
			return
		}
		name := "fontget-test"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		out := filepath.Join(dir, name)
		root, err := filepath.Abs("..")
		if err != nil {
			testBinErr = err
			return
		}
		cmd := exec.Command("go", "build", "-o", out, ".")
		cmd.Dir = root
		b, err := cmd.CombinedOutput()
		if err != nil {
			testBinErr = err
			return
		}
		_ = b
		testBinPath = out
	})
	if testBinErr != nil {
		t.Fatalf("go build: %v", testBinErr)
	}
	return testBinPath
}

func TestDisplayedErrorSkipsDuplicate(t *testing.T) {
	err := shared.AlreadyPrinted(errors.New("shown"))
	var displayed *shared.DisplayedError
	if !errors.As(err, &displayed) {
		t.Fatal("expected DisplayedError")
	}
}

type copyFontManager struct {
	dir        string
	registered map[string]bool
}

func (m *copyFontManager) FlushFontCache(scope platform.InstallationScope) error { return nil }
func (m *copyFontManager) InstallFont(fontPath string, scope platform.InstallationScope, force bool, opts *platform.InstallFontOptions) error {
	if opts == nil {
		opts = &platform.InstallFontOptions{}
	}
	if m.registered == nil {
		m.registered = map[string]bool{}
	}
	dest := filepath.Join(m.dir, filepath.Base(fontPath))
	name := filepath.Base(fontPath)
	mut, err := platform.PlaceFontFile(fontPath, dest, force, opts)
	if err != nil {
		return err
	}
	mut.FontName = name
	mut.Scope = scope
	mut.ResourceRegistered = true
	m.registered[name] = true
	mut.UndoRegistration = func() error {
		delete(m.registered, name)
		return nil
	}
	if opts.Mutation != nil {
		*opts.Mutation = mut
	}
	if opts.FailPoint == platform.InstallFailRegister {
		_ = platform.RollbackMutation(mut)
		return errors.New("injected failure at register")
	}
	return nil
}
func (m *copyFontManager) RemoveFont(fontName string, scope platform.InstallationScope, opts *platform.RemoveFontOptions) error {
	return os.Remove(filepath.Join(m.dir, fontName))
}
func (m *copyFontManager) GetFontDir(scope platform.InstallationScope) string { return m.dir }
func (m *copyFontManager) RequiresElevation(scope platform.InstallationScope) bool {
	return false
}
func (m *copyFontManager) IsElevated() (bool, error) { return true, nil }
func (m *copyFontManager) GetElevationCommand() (string, []string, error) {
	return "", nil, nil
}

func TestInstallKeepsCompletedFileAfterInjectedFailure(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	fm := &copyFontManager{dir: fontDir}

	a := testutil.MinimalTTF("Alpha", "Regular")
	b := testutil.MinimalTTF("Beta", "Regular")
	staging, err := platform.NewOperationStaging()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = staging.Cleanup() })
	pathA := filepath.Join(staging.Root, "Alpha-Regular.ttf")
	pathB := filepath.Join(staging.Root, "Beta-Regular.ttf")
	if err := os.WriteFile(pathA, a, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pathB, b, 0644); err != nil {
		t.Fatal(err)
	}

	installed, _, _, _, _, _, _, err := installDownloadedFonts(context.Background(), []string{pathA, pathB}, fm, platform.UserScope, fontDir, false, nil, &installTestControl{failAfterMutations: 1}, nil)
	if err == nil {
		t.Fatal("expected injected failure")
	}
	if installed != 1 {
		t.Fatalf("expected 1 completed file kept, got %d", installed)
	}
	if _, statErr := os.Stat(filepath.Join(fontDir, "Alpha-Regular.ttf")); statErr != nil {
		t.Fatalf("completed file must remain: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(fontDir, "Beta-Regular.ttf")); !os.IsNotExist(statErr) {
		t.Fatal("second file must not have been installed")
	}
}

func TestInstallForceReplaceNoBackup(t *testing.T) {
	fontDir := t.TempDir()
	old := testutil.MinimalTTF("OldFam", "Regular")
	neu := testutil.MinimalTTF("NewFam", "Regular")
	dst := filepath.Join(fontDir, "Face.ttf")
	if err := os.WriteFile(dst, old, 0644); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(t.TempDir(), "Face.ttf")
	if err := os.WriteFile(src, neu, 0644); err != nil {
		t.Fatal(err)
	}
	mut, err := platform.PlaceFontFile(src, dst, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if mut.BackupPath != "" {
		t.Fatal("force replace must not create a backup")
	}
	got, _ := os.ReadFile(dst)
	if !bytes.Equal(got, neu) {
		t.Fatal("replace did not write new bytes")
	}
	if err := platform.RollbackMutation(mut); err != nil {
		t.Fatal(err)
	}
	got, _ = os.ReadFile(dst)
	if !bytes.Equal(got, neu) {
		t.Fatal("without backup, rollback leaves replaced bytes")
	}
}

func TestPackageFailureDoesNotCountRolledBackAsInstalled(t *testing.T) {
	res := buildInstallResult(InstallStatusFailed, "Installation failed", 0, 0, 1, nil, nil, 0)
	if res.Success != 0 || res.Status != InstallStatusFailed {
		t.Fatalf("%+v", res)
	}
}

func TestInstallKeepsCompletedFilesAfterLaterInjectedFailure(t *testing.T) {
	fontDir := t.TempDir()
	fm := &copyFontManager{dir: fontDir}
	staging, err := platform.NewOperationStaging()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = staging.Cleanup() })
	pathA := filepath.Join(staging.Root, "Alpha-Regular.ttf")
	pathB := filepath.Join(staging.Root, "Beta-Regular.ttf")
	if err := os.WriteFile(pathA, testutil.MinimalTTF("Alpha", "Regular"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pathB, testutil.MinimalTTF("Beta", "Regular"), 0644); err != nil {
		t.Fatal(err)
	}
	installed, _, _, _, _, _, mutations, err := installDownloadedFonts(context.Background(), []string{pathA, pathB}, fm, platform.UserScope, fontDir, false, nil, &installTestControl{failAfterMutations: 2}, nil)
	if err == nil {
		t.Fatal("expected injected failure")
	}
	if installed != 2 || len(mutations) < 2 {
		t.Fatalf("both files should complete before stop: installed=%d mutations=%d", installed, len(mutations))
	}
	for _, name := range []string{"Alpha-Regular.ttf", "Beta-Regular.ttf"} {
		if _, statErr := os.Stat(filepath.Join(fontDir, name)); statErr != nil {
			t.Fatalf("completed file missing %s: %v", name, statErr)
		}
	}
}

func TestInstallRegisterFailRollsBack(t *testing.T) {
	fontDir := t.TempDir()
	fm := &copyFontManager{dir: fontDir}
	staging, err := platform.NewOperationStaging()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = staging.Cleanup() })
	src := filepath.Join(staging.Root, "Face.ttf")
	if err := os.WriteFile(src, testutil.MinimalTTF("Face", "Regular"), 0644); err != nil {
		t.Fatal(err)
	}
	_, _, _, _, _, _, mutations, err := installDownloadedFonts(context.Background(), []string{src}, fm, platform.UserScope, fontDir, false, nil, &installTestControl{failRegister: true}, nil)
	if err == nil {
		t.Fatal("expected register failure")
	}
	if len(mutations) != 0 {
		t.Fatalf("failed file should not remain in mutations: %d", len(mutations))
	}
	entries, _ := os.ReadDir(fontDir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		t.Fatalf("register failure left %s", e.Name())
	}
}

func TestInstallProvenanceFailRollsBackCurrentFile(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	payload := testutil.MinimalTTF("ProvFam", "Regular")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "font/ttf")
		_, _ = w.Write(payload)
	}))
	t.Cleanup(srv.Close)

	fontDir := t.TempDir()
	fm := &copyFontManager{dir: fontDir}
	staging, err := platform.NewOperationStaging()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = staging.Cleanup() })
	files := []repo.FontFile{{
		Name:        "ProvFam",
		Variant:     "Regular",
		Path:        "ProvFam-Regular.ttf",
		DownloadURL: srv.URL + "/ProvFam-Regular.ttf",
	}}
	res, err := installFont(context.Background(), files, "test.prov", fm, platform.UserScope, false, fontDir, staging, true, nil, &installTestControl{failProvenance: true})
	if err == nil {
		t.Fatal("expected provenance failure")
	}
	if res == nil || res.Status == InstallStatusCompleted || res.Success != 0 {
		t.Fatalf("must not report complete after provenance failure: %+v", res)
	}
	entries, _ := os.ReadDir(fontDir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		t.Fatalf("provenance failure left installed file %s", e.Name())
	}
}

// recordingFontManager records remove/install order for force-replace lock regressions.
type recordingFontManager struct {
	*copyFontManager
	ops []string
}

func (m *recordingFontManager) RemoveFont(fontName string, scope platform.InstallationScope, opts *platform.RemoveFontOptions) error {
	m.ops = append(m.ops, "remove:"+fontName)
	return m.copyFontManager.RemoveFont(fontName, scope, opts)
}

func (m *recordingFontManager) InstallFont(fontPath string, scope platform.InstallationScope, force bool, opts *platform.InstallFontOptions) error {
	m.ops = append(m.ops, "install:"+filepath.Base(fontPath))
	return m.copyFontManager.InstallFont(fontPath, scope, force, opts)
}

func TestInstallFontForceReplaceTrackedPackageUnderSingleLock(t *testing.T) {
	// Regression: force must not re-acquire LockDestination while installFont already holds it.
	// Nested lock waits on itself until the short deadline; production lock timeout is 30s.
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()

	oldPayload := testutil.MinimalTTF("OldFace", "Regular")
	newPayload := testutil.MinimalTTF("NewFace", "Bold")
	if bytes.Equal(oldPayload, newPayload) {
		t.Fatal("fixtures must differ")
	}

	existing := filepath.Join(fontDir, "Face.ttf")
	if err := os.WriteFile(existing, oldPayload, 0644); err != nil {
		t.Fatal(err)
	}
	if err := installations.RecordInstallation(installations.RecordParams{
		FontID: "test.force-lock",
		Scope:  "user",
		Files: []installations.InstalledFontFile{
			{Path: existing, SFNT: installations.SFNTSnapshot{Family: "OldFace", Style: "Regular"}},
		},
	}); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "font/ttf")
		_, _ = w.Write(newPayload)
	}))
	t.Cleanup(srv.Close)

	staging, err := platform.NewOperationStaging()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = staging.Cleanup() })

	fm := &recordingFontManager{copyFontManager: &copyFontManager{dir: fontDir}}
	files := []repo.FontFile{{
		Name:        "NewFace",
		Variant:     "Bold",
		Path:        "Face.ttf",
		DownloadURL: srv.URL + "/Face.ttf",
	}}

	// Shorter than the 30s production lock wait so nested-lock regressions fail promptly.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	res, err := installFont(ctx, files, "test.force-lock", fm, platform.UserScope, true, fontDir, staging, true, nil, nil)
	if err != nil {
		t.Fatalf("force replace under single lock: %v", err)
	}
	if res == nil || res.Status != InstallStatusCompleted || res.Success != 1 {
		t.Fatalf("expected completed force install: %+v", res)
	}
	if len(fm.ops) != 2 || !strings.HasPrefix(fm.ops[0], "remove:") || !strings.HasPrefix(fm.ops[1], "install:") {
		t.Fatalf("want remove then install, got %#v", fm.ops)
	}
	if fm.ops[0] != "remove:Face.ttf" {
		t.Fatalf("must remove tracked Face.ttf first, got %q", fm.ops[0])
	}
	installedBase := strings.TrimPrefix(fm.ops[1], "install:")
	if installedBase == "" {
		t.Fatal("missing install basename")
	}

	if _, statErr := os.Stat(existing); !os.IsNotExist(statErr) {
		t.Fatal("old tracked Face.ttf must be gone after force remove")
	}
	dest := filepath.Join(fontDir, installedBase)
	got, readErr := os.ReadFile(dest)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, newPayload) {
		t.Fatal("destination must contain replacement content")
	}

	reg, loadErr := installations.Load()
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	inst := reg.FindByFontID("test.force-lock")
	if inst == nil || inst.IsIncomplete() {
		t.Fatalf("expected complete tracked install: %+v", inst)
	}
	bases := inst.BasenamesForDir(fontDir)
	if len(bases) != 1 || !strings.EqualFold(bases[0], installedBase) {
		t.Fatalf("registry files: %#v want %q", bases, installedBase)
	}

	lockCtx, lockCancel := context.WithTimeout(context.Background(), time.Second)
	defer lockCancel()
	unlock, lockErr := installations.LockDestination(lockCtx, fontDir)
	if lockErr != nil {
		t.Fatalf("destination lock must be free after force install: %v", lockErr)
	}
	unlock()
}

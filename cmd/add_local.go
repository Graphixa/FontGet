package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"fontget/internal/components"
	"fontget/internal/installations"
	"fontget/internal/output"
	"fontget/internal/platform"
	"fontget/internal/repo"
	"fontget/internal/shared"
)

const localInstallConfirmThreshold = 50

const (
	localSourceName       = "Local"
	localBackupSourceName = "FontGet Backup"
)

type addArgKind int

const (
	addArgCatalog addArgKind = iota
	addArgLocal
)

type addArgTarget struct {
	Raw  string
	Kind addArgKind
	Path string // absolute path when Kind == addArgLocal
}

// classifyAddArg treats existing font files, directories, and .zip paths as local install targets.
func classifyAddArg(arg string) addArgTarget {
	arg = strings.TrimSpace(arg)
	info, err := os.Stat(arg)
	if err != nil {
		return addArgTarget{Raw: arg, Kind: addArgCatalog}
	}
	abs, absErr := filepath.Abs(arg)
	if absErr != nil {
		abs = arg
	}
	if info.IsDir() {
		return addArgTarget{Raw: arg, Kind: addArgLocal, Path: abs}
	}
	ext := strings.ToLower(filepath.Ext(abs))
	if ext == ".zip" || platform.IsLocalInstallFontExt(ext) {
		return addArgTarget{Raw: arg, Kind: addArgLocal, Path: abs}
	}
	return addArgTarget{Raw: arg, Kind: addArgCatalog}
}

func classifyAddArgs(args []string) (local []addArgTarget, catalog []string) {
	for _, a := range args {
		t := classifyAddArg(a)
		if t.Kind == addArgLocal {
			local = append(local, t)
		} else {
			catalog = append(catalog, t.Raw)
		}
	}
	return local, catalog
}

func localFontID(family string) string {
	key := repo.FontKey(family)
	if key == "" {
		key = "unknown"
	}
	return "local." + key
}

type localFontGroup struct {
	FontID     string
	FamilyName string
	SourceName string
	Candidates []platform.LocalFontCandidate
}

type localInstallOutcome struct {
	Installed int
	Skipped   int
	Failed    int
	Dupes     int
	Conflicts int
	Errors    []string
	Groups    int
}

// installLocalPath discovers, confirms, and installs fonts from a local file/folder/zip
// using the same progress-bar UX as catalog add.
func installLocalPath(
	ctx context.Context,
	path string,
	fontManager platform.FontManager,
	installScope platform.InstallationScope,
	fontDir string,
	force bool,
	yes bool,
	verbose bool,
	debug bool,
) (*localInstallOutcome, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	res, err := platform.DiscoverAndDedupeLocalFonts(path)
	if err != nil {
		return nil, err
	}
	for _, w := range res.Warnings {
		output.GetVerbose().Warning("%s", w)
		output.GetDebug().State("local ingest: %s", w)
	}
	if res.FontGetBackup {
		output.GetVerbose().Info("FontGet backup detected")
		output.GetDebug().State("FontGet backup zip comment present on %s", path)
	}

	out := &localInstallOutcome{
		Dupes:     res.HashDupesSkipped + res.FaceDupesSkipped,
		Conflicts: res.ConflictsSkipped,
	}

	kept := res.Kept
	if len(kept) == 0 {
		return out, fmt.Errorf("no font files to install after deduplication")
	}

	if len(kept) >= localInstallConfirmThreshold && !yes {
		msg := fmt.Sprintf(
			"Found %d font file(s) (%d duplicates skipped, %d from nested archives, %d filename conflicts skipped).\nInstall to %s scope?",
			len(kept), out.Dupes, res.NestedZipFonts, res.ConflictsSkipped, installScope,
		)
		if !components.UseInteractiveRenderer() {
			return out, fmt.Errorf("installing %d local fonts requires confirmation; re-run with --yes", len(kept))
		}
		ok, cerr := components.RunConfirm("Install local fonts?", msg)
		if cerr != nil {
			return out, cerr
		}
		if !ok {
			return out, shared.AlreadyPrinted(fmt.Errorf("local install cancelled"))
		}
	}

	sourceName := localSourceName
	installSrc := "local"
	if res.FontGetBackup {
		sourceName = localBackupSourceName
		installSrc = "fontget-backup"
	}

	groups := buildLocalFontGroups(kept, sourceName)
	out.Groups = len(groups)
	if len(groups) == 0 {
		return out, fmt.Errorf("no font files to install after deduplication")
	}

	operationItems := setupLocalInstallationProgressBar(groups)

	title := OpInstallingFonts
	if installScope == platform.MachineScope {
		title = OpInstallingFontsAllUsers
	}

	if !output.IsVerboseOutputEnabled() {
		fmt.Println()
	}

	cancelled := false
	progressErr := components.RunProgressBar(
		title,
		operationItems,
		verbose,
		debug,
		func(send func(msg tea.Msg), cancelChan <-chan struct{}) error {
			opCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			go func() {
				select {
				case <-cancelChan:
					cancel()
				case <-opCtx.Done():
				}
			}()

			unlockDest, lockErr := installations.LockDestination(opCtx, fontDir)
			if lockErr != nil {
				return lockErr
			}
			defer unlockDest()

			for itemIndex, group := range groups {
				if err := opCtx.Err(); err != nil {
					cancelled = true
					return shared.ErrOperationCancelled
				}

				send(components.ItemUpdateMsg{
					Index:   itemIndex,
					Status:  "in_progress",
					Message: InstallingFromLocalMessage(group.SourceName),
				})
				send(components.ProgressUpdateMsg{Percent: float64(itemIndex) / float64(len(groups)) * 100})

				result, ierr := installLocalFontGroup(
					opCtx,
					group,
					fontManager,
					installScope,
					fontDir,
					force,
					installSrc,
					func(u ProgressUpdate) {
						pct := OverallWorkPercent(itemIndex, len(groups), u)
						if msg := ProgressActivityLabel(u, group.SourceName); msg != "" {
							send(components.ItemUpdateMsg{
								Index:   itemIndex,
								Status:  "in_progress",
								Message: msg,
							})
						}
						send(components.ProgressUpdateMsg{Percent: pct})
					},
				)
				if ierr != nil {
					if IsCancelErr(ierr) {
						cancelled = true
						if result != nil {
							out.Installed += result.Success
							out.Skipped += result.Skipped
							out.Failed += result.Failed
							out.Errors = append(out.Errors, result.Errors...)
						}
						send(components.ItemUpdateMsg{
							Index:   itemIndex,
							Status:  InstallStatusFailed,
							Message: "Cancelled",
						})
						return shared.ErrOperationCancelled
					}
					if result != nil {
						out.Installed += result.Success
						out.Skipped += result.Skipped
						out.Failed += result.Failed
						out.Errors = append(out.Errors, result.Errors...)
					} else {
						out.Failed++
						out.Errors = append(out.Errors, ierr.Error())
					}
					send(components.ItemUpdateMsg{
						Index:        itemIndex,
						Status:       InstallStatusFailed,
						Message:      "Operation failed",
						ErrorMessage: ierr.Error(),
					})
					continue
				}

				out.Installed += result.Success
				out.Skipped += result.Skipped
				out.Failed += result.Failed
				out.Errors = append(out.Errors, result.Errors...)

				finalStatus := result.Status
				var errorMsg string
				if finalStatus == InstallStatusFailed && len(result.Errors) > 0 {
					errorMsg = result.Errors[0]
				}
				scopeLabel := InstallScopeLabelUser
				if installScope == platform.MachineScope {
					scopeLabel = InstallScopeLabelMachine
				}
				var variantsWithStatus []string
				if verbose {
					variantsWithStatus = localVariantLines(group.Candidates)
				}
				send(components.ItemUpdateMsg{
					Index:        itemIndex,
					Status:       finalStatus,
					Message:      "Installed",
					ErrorMessage: errorMsg,
					Variants:     variantsWithStatus,
					Scope:        scopeLabel,
				})
				send(components.ProgressUpdateMsg{Percent: OverallWorkPercent(itemIndex, len(groups), ProgressUpdate{Phase: installStepCompleted})})
			}
			return nil
		},
	)

	if progressErr != nil {
		if errorsIsCancel(progressErr) || cancelled {
			return out, shared.ErrOperationCancelled
		}
		return out, progressErr
	}
	return out, nil
}

func errorsIsCancel(err error) bool {
	return err != nil && (IsCancelErr(err) || err == shared.ErrOperationCancelled)
}

func buildLocalFontGroups(kept []platform.LocalFontCandidate, sourceName string) []localFontGroup {
	groups := map[string]*localFontGroup{}
	order := []string{}
	for _, c := range kept {
		id := localFontID(c.Family)
		g, ok := groups[id]
		if !ok {
			familyName := c.Family
			if familyName == "" {
				familyName = id
			}
			g = &localFontGroup{
				FontID:     id,
				FamilyName: familyName,
				SourceName: sourceName,
			}
			groups[id] = g
			order = append(order, id)
		}
		g.Candidates = append(g.Candidates, c)
	}
	out := make([]localFontGroup, 0, len(order))
	for _, id := range order {
		out = append(out, *groups[id])
	}
	return out
}

func setupLocalInstallationProgressBar(groups []localFontGroup) []components.OperationItem {
	items := make([]components.OperationItem, 0, len(groups))
	for _, g := range groups {
		variants := make([]string, 0, len(g.Candidates))
		for _, c := range g.Candidates {
			if style := strings.TrimSpace(c.Style); style != "" {
				variants = append(variants, style)
			} else {
				variants = append(variants, c.Basename)
			}
		}
		items = append(items, components.OperationItem{
			Name:          g.FamilyName,
			SourceName:    g.SourceName,
			Status:        "pending",
			StatusMessage: "Pending",
			Variants:      variants,
			Scope:         "",
		})
	}
	return items
}

func localVariantLines(cands []platform.LocalFontCandidate) []string {
	lines := make([]string, 0, len(cands))
	for _, c := range cands {
		style := strings.TrimSpace(c.Style)
		if style == "" {
			lines = append(lines, c.Basename)
			continue
		}
		if fam := strings.TrimSpace(c.Family); fam != "" {
			lines = append(lines, strings.TrimSpace(fam+" "+humanizeFontStyleLabel(style)))
			continue
		}
		lines = append(lines, humanizeFontStyleLabel(style))
	}
	return lines
}

// InstallingFromLocalMessage is the initial activity label for a local family install.
func InstallingFromLocalMessage(sourceName string) string {
	sourceName = strings.TrimSpace(sourceName)
	if sourceName == "" {
		return "Installing..."
	}
	return "Installing from " + sourceName + "..."
}

func installLocalFontGroup(
	ctx context.Context,
	group localFontGroup,
	fontManager platform.FontManager,
	installScope platform.InstallationScope,
	fontDir string,
	force bool,
	installSrc string,
	onProgress ProgressFunc,
) (*InstallResult, error) {
	if force {
		existing := packageBasenamesFromRegistry(group.FontID, fontDir)
		if len(existing) > 0 {
			forceProgress := onProgress
			if onProgress != nil {
				forceProgress = func(u ProgressUpdate) {
					onProgress(remapForceInstallProgress(u))
				}
				forceProgress(ProgressUpdate{Phase: installStepForceRemove, Kind: ProgressCount, Done: 0, Total: float64(len(existing))})
			}
			_, _, remFailed, _, remErrs, remErr := removeFontFiles(RemoveFontFilesParams{
				Ctx:                  ctx,
				MatchingFonts:        existing,
				FontManager:          fontManager,
				Scope:                installScope,
				FontDir:              fontDir,
				FontID:               group.FontID,
				IsCriticalSystemFont: shared.IsCriticalSystemFont,
				OnProgress:           forceProgress,
			})
			if remErr != nil || remFailed > 0 {
				msg := "Force removal failed"
				if IsCancelErr(remErr) {
					msg = msgForceRemovalCancelledShort
				}
				res := buildInstallResult(InstallStatusFailed, msg, 0, 0, remFailed, existing, remErrs, 0)
				if remErr != nil {
					return res, remErr
				}
				return res, fmt.Errorf("force install: removal incomplete for %s", group.FontID)
			}
		}
	}

	basenames := make([]string, 0, len(group.Candidates))
	for _, c := range group.Candidates {
		basenames = append(basenames, c.Basename)
	}
	track := newInstallTracker(group.FontID, nil, installScope, fontDir, basenames)
	track.catalogName = group.FamilyName
	track.installSrc = installSrc

	paths := make([]string, 0, len(group.Candidates))
	var tmpDirs []string
	defer func() {
		for _, d := range tmpDirs {
			_ = os.RemoveAll(d)
		}
	}()

	for _, c := range group.Candidates {
		if err := ctx.Err(); err != nil {
			return buildInstallResult(InstallStatusFailed, msgInstallationCancelledShort, 0, 0, 0, nil, nil, 0), err
		}
		if c.FromZip {
			if onProgress != nil {
				onProgress(ProgressUpdate{Phase: installStepExtract, Kind: ProgressFlag, Done: 0, Total: 1})
			}
			p, xerr := platform.ExtractZipEntryToTemp(c.ZipPath, c.ZipEntry, c.Basename)
			if xerr != nil {
				res := buildInstallResult(InstallStatusFailed, "Extract failed", 0, 0, 1, nil, []string{fmt.Sprintf("%s: %v", c.Basename, xerr)}, 0)
				return res, xerr
			}
			paths = append(paths, p)
			tmpDirs = append(tmpDirs, filepath.Dir(p))
			if onProgress != nil {
				onProgress(ProgressUpdate{Phase: installStepExtract, Kind: ProgressFlag, Done: 1, Total: 1})
			}
			continue
		}
		paths = append(paths, c.DiskPath)
	}

	installed, skipped, failed, details, errs, downloadSize, _, ierr := installDownloadedFonts(
		ctx, paths, fontManager, installScope, fontDir, force, onProgress, nil, track, false,
	)
	if ierr != nil {
		status := InstallStatusFailed
		message := "Installation failed"
		if IsCancelErr(ierr) {
			message = msgInstallationCancelledShort
		}
		return buildInstallResult(status, message, installed, skipped, failed, details, errs, downloadSize), ierr
	}
	if failed > 0 {
		res := buildInstallResult(InstallStatusFailed, "Installation failed", installed, skipped, failed, details, errs, downloadSize)
		return res, fmt.Errorf("package install incomplete")
	}
	status := InstallStatusCompleted
	message := "Installed"
	if skipped > 0 && installed == 0 && failed == 0 {
		status = InstallStatusSkipped
		message = "Already installed"
	}
	return buildInstallResult(status, message, installed, skipped, failed, details, errs, downloadSize), nil
}

package cmd

import (
	"context"
	"errors"
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
	if ext == ".zip" || platform.IsLocalInstallFontExt(ext) || platform.IsLocalCollectionFontExt(ext) {
		return addArgTarget{Raw: arg, Kind: addArgLocal, Path: abs}
	}
	return addArgTarget{Raw: arg, Kind: addArgCatalog}
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
	InstallSrc string
	Candidates []platform.LocalFontCandidate
	RetryPaths []string // original user args that contributed to this family
}

type localPrepareResult struct {
	Groups    []localFontGroup
	Dupes     int
	Conflicts int
	Path      string
}

// prepareLocalInstall discovers, dedupes, and optionally confirms a local path.
// It does not install; callers add groups to a unified progress session.
func prepareLocalInstall(path, rawArg string, installScope platform.InstallationScope, yes bool) (*localPrepareResult, error) {
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

	out := &localPrepareResult{
		Dupes:     res.HashDupesSkipped + res.FaceDupesSkipped,
		Conflicts: res.ConflictsSkipped,
		Path:      path,
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
	out.Groups = buildLocalFontGroups(kept, sourceName, installSrc)
	retry := strings.TrimSpace(rawArg)
	if retry == "" {
		retry = path
	}
	for i := range out.Groups {
		out.Groups[i].RetryPaths = []string{retry}
	}
	if len(out.Groups) == 0 {
		return out, fmt.Errorf("no font files to install after deduplication")
	}
	return out, nil
}

// mergeLocalWorkItems merges local groups that share a Font ID into one work item each,
// preserving first-seen family order and catalog item positions. Candidates are deduped after merge.
func mergeLocalWorkItems(items []addWorkItem) ([]addWorkItem, error) {
	if len(items) == 0 {
		return items, nil
	}
	out := make([]addWorkItem, 0, len(items))
	localAt := make(map[string]int) // FontID -> index in out
	for _, it := range items {
		if it.Kind != addWorkLocal || it.Local == nil {
			out = append(out, it)
			continue
		}
		g := *it.Local
		if idx, ok := localAt[g.FontID]; ok {
			existing := out[idx].Local
			existing.Candidates = append(existing.Candidates, g.Candidates...)
			existing.RetryPaths = append(existing.RetryPaths, g.RetryPaths...)
			if existing.SourceName == localSourceName && g.SourceName == localBackupSourceName {
				existing.SourceName = g.SourceName
				existing.InstallSrc = g.InstallSrc
			}
			continue
		}
		copied := g
		localAt[g.FontID] = len(out)
		out = append(out, addWorkItem{Kind: addWorkLocal, Local: &copied})
	}
	for _, it := range out {
		if it.Kind != addWorkLocal || it.Local == nil {
			continue
		}
		kept, _, _, _, warnings, err := platform.DedupeLocalCandidates(it.Local.Candidates)
		if err != nil {
			return nil, err
		}
		for _, w := range warnings {
			output.GetVerbose().Warning("%s", w)
		}
		it.Local.Candidates = kept
		it.Local.RetryPaths = dedupeExactStrings(it.Local.RetryPaths)
		if len(it.Local.Candidates) == 0 {
			return nil, fmt.Errorf("no font files to install after deduplication")
		}
	}
	if err := validateLocalDestinationCollisions(out); err != nil {
		return nil, err
	}
	return out, nil
}

// validateLocalDestinationCollisions rejects EqualFold basename clashes across different local families.
func validateLocalDestinationCollisions(items []addWorkItem) error {
	type hit struct {
		family string
		label  string
	}
	seen := map[string]hit{}
	for _, it := range items {
		if it.Kind != addWorkLocal || it.Local == nil {
			continue
		}
		for _, c := range it.Local.Candidates {
			key := strings.ToLower(c.Basename)
			label := platform.CandidateLabel(c)
			if prev, ok := seen[key]; ok {
				return fmt.Errorf(
					"destination filename collision: %s (%s) and %s (%s) both install as %s",
					prev.label, prev.family, label, it.Local.FamilyName, c.Basename,
				)
			}
			seen[key] = hit{family: it.Local.FamilyName, label: label}
		}
	}
	return nil
}

// stageLocalWorkItems copies/extracts every local candidate into staging before force removal.
func stageLocalWorkItems(items []addWorkItem, staging *platform.OperationStaging) error {
	if staging == nil {
		return fmt.Errorf("operation staging is required for local install")
	}
	for _, it := range items {
		if it.Kind != addWorkLocal || it.Local == nil {
			continue
		}
		staged := make([]platform.LocalFontCandidate, 0, len(it.Local.Candidates))
		for _, c := range it.Local.Candidates {
			sc, err := platform.StageLocalCandidate(staging, c)
			if err != nil {
				return fmt.Errorf("stage local font %s: %w", platform.CandidateLabel(c), err)
			}
			staged = append(staged, sc)
		}
		it.Local.Candidates = staged
	}
	return nil
}

type addWorkKind int

const (
	addWorkLocal addWorkKind = iota
	addWorkCatalog
)

// addWorkItem is one row in a unified add progress session.
type addWorkItem struct {
	Kind    addWorkKind
	Local   *localFontGroup
	Catalog *FontToInstall
}

func buildLocalFontGroups(kept []platform.LocalFontCandidate, sourceName, installSrc string) []localFontGroup {
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
				InstallSrc: installSrc,
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

func setupUnifiedAddProgressBar(items []addWorkItem) []components.OperationItem {
	out := make([]components.OperationItem, 0, len(items))
	for _, it := range items {
		switch it.Kind {
		case addWorkLocal:
			g := it.Local
			variants := make([]string, 0, len(g.Candidates))
			for _, c := range g.Candidates {
				if style := strings.TrimSpace(c.Style); style != "" {
					variants = append(variants, style)
				} else {
					variants = append(variants, c.Basename)
				}
			}
			out = append(out, components.OperationItem{
				Name:          g.FamilyName,
				SourceName:    g.SourceName,
				Status:        "pending",
				StatusMessage: "Pending",
				Variants:      variants,
			})
		case addWorkCatalog:
			fg := it.Catalog
			fontName := fg.FontName
			if len(fg.Fonts) > 0 && fg.Fonts[0].Name != "" {
				fontName = fg.Fonts[0].Name
			}
			var variantNames []string
			for _, f := range fg.Fonts {
				variantNames = append(variantNames, f.Variant)
			}
			out = append(out, components.OperationItem{
				Name:          fontName,
				SourceName:    fg.SourceName,
				Status:        "pending",
				StatusMessage: "Pending",
				Variants:      variantNames,
			})
		}
	}
	return out
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

// addSessionCompletion is the terminal state of a unified add session.
type addSessionCompletion int

const (
	addCompleted addSessionCompletion = iota
	addCancelledRemaining
	addCancelledDone
)

func recordItemFailure(status *InstallationStatus, result *InstallResult, err error) {
	status.FailedItems++
	if result != nil {
		status.Installed += result.Success
		status.Skipped += result.Skipped
		status.Failed += result.Failed
		status.Errors = append(status.Errors, result.Errors...)
	}
	if err != nil {
		msg := err.Error()
		found := false
		for _, e := range status.Errors {
			if e == msg {
				found = true
				break
			}
		}
		if !found {
			status.Errors = append(status.Errors, msg)
		}
	}
}

type addCancelRetry struct {
	CatalogIDs   []string
	LocalPaths   []string
	MissingLocal []string
}

func collectCancelRetry(items []addWorkItem, fromIndex int) addCancelRetry {
	var r addCancelRetry
	for j := fromIndex; j < len(items); j++ {
		switch items[j].Kind {
		case addWorkCatalog:
			if items[j].Catalog != nil {
				r.CatalogIDs = append(r.CatalogIDs, items[j].Catalog.FontID)
			}
		case addWorkLocal:
			if items[j].Local == nil {
				continue
			}
			for _, p := range items[j].Local.RetryPaths {
				if _, err := os.Stat(p); err == nil {
					r.LocalPaths = append(r.LocalPaths, p)
				} else {
					r.MissingLocal = append(r.MissingLocal, p)
				}
			}
		}
	}
	r.CatalogIDs = dedupePackageIDs(r.CatalogIDs)
	r.LocalPaths = dedupeExactStrings(r.LocalPaths)
	r.MissingLocal = dedupeExactStrings(r.MissingLocal)
	return r
}

func finishAddCancel(retry addCancelRetry, scope string, force bool, workRemaining bool) error {
	if !workRemaining {
		return nil
	}
	text := FormatInstallationCancelledTextMixed(retry.CatalogIDs, retry.LocalPaths, retry.MissingLocal, scope, force)
	printCancelledText(text)
	return shared.AlreadyPrinted(shared.ErrOperationCancelled)
}

// runUnifiedAddSession runs one progress bar for mixed local + catalog work items.
// staging may be nil; catalog installs create one when needed via installFont.
// Local candidates must already be staged when present.
func runUnifiedAddSession(
	ctx context.Context,
	items []addWorkItem,
	fontManager platform.FontManager,
	installScope platform.InstallationScope,
	fontDir string,
	force bool,
	verbose bool,
	debug bool,
	staging *platform.OperationStaging,
) (*InstallationStatus, addSessionCompletion, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	status := &InstallationStatus{Details: make([]string, 0)}
	if len(items) == 0 {
		return status, addCompleted, nil
	}

	operationItems := setupUnifiedAddProgressBar(items)
	title := OpInstallingFonts
	if installScope == platform.MachineScope {
		title = OpInstallingFontsAllUsers
	}
	if !output.IsVerboseOutputEnabled() {
		fmt.Println()
	}

	suppressVerboseDownloads := components.UseInteractiveRenderer() && !debug
	cancelled := false
	var recoveryFailure *localRecoveryError
	cancelFrom := -1 // first unfinished item index when cancelled; -1 = after all items

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

			total := len(items)
			for itemIndex, item := range items {
				if err := opCtx.Err(); err != nil {
					cancelled = true
					cancelFrom = itemIndex
					return shared.ErrOperationCancelled
				}

				switch item.Kind {
				case addWorkLocal:
					group := item.Local
					send(components.ItemUpdateMsg{
						Index:   itemIndex,
						Status:  "in_progress",
						Message: InstallingFromLocalMessage(group.SourceName),
					})
					send(components.ProgressUpdateMsg{Percent: float64(itemIndex) / float64(total) * 100})

					var th progressThrottle
					onProgress := func(u ProgressUpdate) {
						pct := OverallWorkPercent(itemIndex, total, u)
						if !th.ShouldSend(u, pct) {
							return
						}
						if msg := ProgressActivityLabel(u, group.SourceName); msg != "" {
							send(components.ItemUpdateMsg{
								Index:   itemIndex,
								Status:  "in_progress",
								Message: msg,
							})
						}
						send(components.ProgressUpdateMsg{Percent: pct})
					}

					result, ierr := installLocalFontGroup(opCtx, *group, fontManager, installScope, fontDir, force, group.InstallSrc, onProgress, staging)
					if ierr != nil {
						if errors.As(ierr, &recoveryFailure) {
							recordItemFailure(status, result, ierr)
							return ierr
						}
						if IsCancelErr(ierr) {
							cancelled = true
							cancelFrom = itemIndex
							if result != nil {
								status.Installed += result.Success
								status.Skipped += result.Skipped
								status.Failed += result.Failed
							}
							send(components.ItemUpdateMsg{Index: itemIndex, Status: InstallStatusFailed, Message: "Cancelled"})
							return shared.ErrOperationCancelled
						}
						recordItemFailure(status, result, ierr)
						send(components.ItemUpdateMsg{
							Index:        itemIndex,
							Status:       InstallStatusFailed,
							Message:      "Operation failed",
							ErrorMessage: ierr.Error(),
						})
						continue
					}
					status.Installed += result.Success
					status.Skipped += result.Skipped
					status.Failed += result.Failed
					status.Errors = append(status.Errors, result.Errors...)

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
					send(components.ProgressUpdateMsg{Percent: OverallWorkPercent(itemIndex, total, ProgressUpdate{Phase: installStepCompleted})})

				case addWorkCatalog:
					fontGroup := item.Catalog
					send(components.ItemUpdateMsg{
						Index:   itemIndex,
						Status:  "in_progress",
						Message: DownloadFromSourceMessage(fontGroup.SourceName),
					})
					send(components.ProgressUpdateMsg{Percent: float64(itemIndex) / float64(total) * 100})

					var th progressThrottle
					onProgress := func(u ProgressUpdate) {
						pct := OverallWorkPercent(itemIndex, total, u)
						if !th.ShouldSend(u, pct) {
							return
						}
						if msg := ProgressActivityLabel(u, fontGroup.SourceName); msg != "" {
							send(components.ItemUpdateMsg{
								Index:   itemIndex,
								Status:  "in_progress",
								Message: msg,
							})
						}
						send(components.ProgressUpdateMsg{Percent: pct})
					}

					result, err := installFont(
						opCtx,
						fontGroup.Fonts,
						fontGroup.FontID,
						fontManager,
						installScope,
						force,
						fontDir,
						staging,
						suppressVerboseDownloads,
						onProgress,
						nil,
					)
					if err != nil {
						if IsCancelErr(err) {
							cancelled = true
							cancelFrom = itemIndex
							if result != nil {
								status.Installed += result.Success
								status.Skipped += result.Skipped
								status.Failed += result.Failed
							}
							send(components.ItemUpdateMsg{Index: itemIndex, Status: InstallStatusFailed, Message: "Cancelled"})
							return shared.ErrOperationCancelled
						}
						recordItemFailure(status, result, err)
						send(components.ItemUpdateMsg{
							Index:        itemIndex,
							Status:       InstallStatusFailed,
							Message:      "Operation failed",
							ErrorMessage: err.Error(),
						})
						continue
					}

					status.Installed += result.Success
					status.Skipped += result.Skipped
					status.Failed += result.Failed
					status.Errors = append(status.Errors, result.Errors...)

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
						variantsWithStatus = variantLinesForVerboseProgress(fontGroup.Fonts)
					}
					send(components.ItemUpdateMsg{
						Index:        itemIndex,
						Status:       finalStatus,
						Message:      "Installed",
						ErrorMessage: errorMsg,
						Variants:     variantsWithStatus,
						Scope:        scopeLabel,
					})
					send(components.ProgressUpdateMsg{Percent: OverallWorkPercent(itemIndex, total, ProgressUpdate{Phase: installStepCompleted})})
				}
			}
			return nil
		},
	)

	// The interactive renderer may suppress worker messages after cancellation.
	// Print recovery failures only after it has joined the worker and closed.
	if recoveryFailure != nil {
		fmt.Println(recoveryFailure.Error())
		return status, addCompleted, shared.AlreadyPrinted(recoveryFailure)
	}
	if progressErr != nil {
		if errorsIsCancel(progressErr) || cancelled {
			workRemaining := cancelFrom >= 0 && cancelFrom < len(items)
			var retry addCancelRetry
			if workRemaining {
				retry = collectCancelRetry(items, cancelFrom)
			}
			err := finishAddCancel(retry, string(installScope), force, workRemaining)
			if workRemaining {
				return status, addCancelledRemaining, err
			}
			return status, addCancelledDone, nil
		}
		return status, addCompleted, progressErr
	}
	if cancelled {
		workRemaining := cancelFrom >= 0 && cancelFrom < len(items)
		var retry addCancelRetry
		if workRemaining {
			retry = collectCancelRetry(items, cancelFrom)
		}
		err := finishAddCancel(retry, string(installScope), force, workRemaining)
		if workRemaining {
			return status, addCancelledRemaining, err
		}
		return status, addCancelledDone, nil
	}
	return status, addCompleted, nil
}

func errorsIsCancel(err error) bool {
	return err != nil && (IsCancelErr(err) || err == shared.ErrOperationCancelled)
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
	staging *platform.OperationStaging,
) (*InstallResult, error) {
	unlockDest, lockErr := installations.LockDestination(ctx, fontDir)
	if lockErr != nil {
		return buildInstallResult(InstallStatusFailed, "Failed to lock destination", 0, 0, 0, nil, nil, 0), lockErr
	}
	defer unlockDest()

	overlaps := localOverlapsForFontDir(group.Candidates, fontDir)
	prior := snapshotInstallation(group.FontID)
	mutatedOverlap := false

	finishWithRecovery := func(res *InstallResult, opErr error) (*InstallResult, error) {
		if opErr == nil {
			return res, nil
		}
		needRecover := mutatedOverlap
		if !needRecover {
			for _, o := range overlaps {
				if _, err := os.Stat(o.Origin); os.IsNotExist(err) {
					needRecover = true
					break
				}
			}
		}
		if !needRecover || len(overlaps) == 0 {
			return res, opErr
		}
		if rerr := recoverLocalOverlaps(overlaps, fontManager, installScope, fontDir, group, prior, opErr); rerr != nil {
			return res, wrapLocalRecoveryError(opErr, staging, rerr)
		}
		return res, opErr
	}

	if force {
		existing := packageBasenamesFromRegistry(group.FontID, fontDir)
		if len(existing) > 0 {
			if overlapBasenameHit(overlaps, existing) {
				mutatedOverlap = true
			}
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
					return finishWithRecovery(res, remErr)
				}
				return finishWithRecovery(res, fmt.Errorf("force install: removal incomplete for %s", group.FontID))
			}
		}
	}

	basenames := make([]string, 0, len(group.Candidates))
	paths := make([]string, 0, len(group.Candidates))
	for _, c := range group.Candidates {
		if err := ctx.Err(); err != nil {
			res := buildInstallResult(InstallStatusFailed, msgInstallationCancelledShort, 0, 0, 0, nil, nil, 0)
			return finishWithRecovery(res, err)
		}
		basenames = append(basenames, c.Basename)
		paths = append(paths, c.DiskPath)
	}
	// Installing into fontDir with the same basename as an overlapping origin mutates that input.
	for _, o := range overlaps {
		dest := filepath.Join(fontDir, o.Base)
		if strings.EqualFold(platform.CanonicalPath(dest), platform.CanonicalPath(o.Origin)) {
			mutatedOverlap = true
			break
		}
	}
	track := newInstallTracker(group.FontID, nil, installScope, fontDir, basenames)
	track.catalogName = group.FamilyName
	track.installSrc = installSrc

	installed, skipped, failed, details, errs, downloadSize, _, ierr := installDownloadedFonts(
		ctx, paths, fontManager, installScope, fontDir, force, onProgress, nil, track, false,
	)
	if ierr != nil {
		status := InstallStatusFailed
		message := "Installation failed"
		if IsCancelErr(ierr) {
			message = msgInstallationCancelledShort
		}
		res := buildInstallResult(status, message, installed, skipped, failed, details, errs, downloadSize)
		return finishWithRecovery(res, ierr)
	}
	if failed > 0 {
		res := buildInstallResult(InstallStatusFailed, "Installation failed", installed, skipped, failed, details, errs, downloadSize)
		return finishWithRecovery(res, fmt.Errorf("package install incomplete"))
	}
	status := InstallStatusCompleted
	message := "Installed"
	if skipped > 0 && installed == 0 && failed == 0 {
		status = InstallStatusSkipped
		message = "Already installed"
	}
	return buildInstallResult(status, message, installed, skipped, failed, details, errs, downloadSize), nil
}

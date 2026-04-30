package cmd

import (
	"fmt"
	"strings"
	"time"

	"fontget/internal/components"
	"fontget/internal/platform"
	"fontget/internal/repo"
	"fontget/internal/shared"
	"fontget/internal/ui"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Search card: fixed width so the field does not stretch with the terminal.
// Outer width includes rounded border; inner text input width is derived in syncSearchInputWidth.
const (
	browsePageTitle = "Font Browser"

	// Horizontal inset for search, table, and help (not applied to the page title).
	browseWindowHMargin = 1

	browseSearchCardOuterWidth = 52
	browseSearchCardOuterMin   = 28
	// Spaces inside the middle row after the left │ before the prompt (keep small so │ sits tight to >).
	browseSearchInnerLeftPad = 1

	// browseShowFullCatalogInitially fills the results table with every font (respecting the category
	// filter) before the user types a search. Set false to keep the table empty until the first query.
	browseShowFullCatalogInitially = true

	// browseCatalogUsesSearchSortSettings: when true, empty-query browse catalog uses the same sort
	// rules as search (user config, e.g. Search.EnablePopularitySort). When false, alphabetical only.
	browseCatalogUsesSearchSortSettings = true
)

type searchTickMsg struct {
	gen int
}

type installFinishedMsg struct {
	err    error
	result *InstallResult
	fontID string
}

type uninstallFinishedMsg struct {
	err    error
	result *RemoveResult
	fontID string
}

type browseOpProgressMsg struct {
	phase   string
	percent float64 // 0..100; negative means "indeterminate"
}

type browseModalState struct {
	fontID  string
	font    repo.FontInfo
	buttons *components.ButtonGroup
}

// browseToastPopup is a dismissible status message (Enter / Esc).
type browseToastPopup struct {
	body string
}

// browseResultModalState is an acknowledgement dialog after install/remove (RenderDialog + OK).
type browseResultModalState struct {
	borderTitle string
	errorTitle  bool
	body        string
	buttons     *components.ButtonGroup
}

type browseModel struct {
	width  int
	height int

	searchInput textinput.Model
	table       *components.TableModel
	results     []repo.SearchResult

	repository   *repo.Repository
	fontManager  platform.FontManager
	installScope platform.InstallationScope
	fontDir      string
	force        bool

	// Cached manifest categories (sorted); slot 0 = All, 1..n = browseCategoryNames[i-1].
	browseCategoryNames []string
	browseFilterSlot    int

	modal *browseModalState

	resultModal *browseResultModalState

	toastPopup *browseToastPopup

	installing           bool
	installPopupFontName string
	installPopupSource   string

	removing            bool
	removingFontName    string
	removingSourceLabel string

	// Indeterminate-style progress (0–100) for install/remove status overlay.
	statusProgress float64
	statusPhase    string

	opMsgCh <-chan tea.Msg

	debounceGen int

	tableViewportH int

	// Cached total fonts in the manifest (for footer “shown / total”).
	browseManifestFontCount int
}

func newBrowseModel(
	repository *repo.Repository,
	fontManager platform.FontManager,
	installScope platform.InstallationScope,
	fontDir string,
	force bool,
) (*browseModel, error) {
	ti := textinput.New()
	ti.Placeholder = "Search fonts…"
	ti.Focus()
	ti.CharLimit = 200
	ti.TextStyle = ui.FormInput
	ti.PlaceholderStyle = ui.FormPlaceholder
	// Default until WindowSize; inner row = outer − 2 (│) − left padding inside row.
	ti.Width = max(8, browseSearchCardOuterWidth-2-browseSearchInnerLeftPad)

	tc := components.TableConfig{
		Columns: []components.ColumnConfig{
			// Prefer keeping font names readable as the terminal narrows.
			// Lower Priority number = preserved more by calculateColumnWidths.
			{Header: "Name", Truncatable: true, MinWidth: 22, PercentWidth: 32, Priority: 1},
			{Header: "Font ID", Truncatable: true, MinWidth: 10, PercentWidth: 28, Priority: 4},
			{Header: "License", MaxWidth: 18, Truncatable: true, MinWidth: 6, PercentWidth: 8, Priority: 6},
			{Header: "Categories", Truncatable: true, MinWidth: 6, PercentWidth: 18, Priority: 7},
			{Header: "Source", Truncatable: true, MinWidth: 6, PercentWidth: 14, Priority: 8},
		},
		Padding: 0,
		Rows:    [][]string{},
		Height:  10,
		Mode:    components.TableModeDynamic,
	}
	tm, err := components.NewTableModel(tc)
	if err != nil {
		return nil, err
	}
	tm.SetFocus(false)

	return &browseModel{
		searchInput:             ti,
		table:                   tm,
		repository:              repository,
		fontManager:             fontManager,
		installScope:            installScope,
		fontDir:                 fontDir,
		force:                   force,
		browseCategoryNames:     repository.GetAllCategories(),
		browseManifestFontCount: repository.TotalManifestFonts(),
		tableViewportH:          8,
	}, nil
}

func (m *browseModel) Init() tea.Cmd {
	if browseShowFullCatalogInitially {
		return tea.Batch(textinput.Blink, m.applySearch())
	}
	return textinput.Blink
}

func (m *browseModel) showToastPopup(body string) {
	m.toastPopup = &browseToastPopup{body: body}
}

func (m *browseModel) dismissToastPopup() {
	m.toastPopup = nil
}

func (m *browseModel) scheduleSearch() tea.Cmd {
	m.debounceGen++
	gen := m.debounceGen
	return tea.Tick(280*time.Millisecond, func(time.Time) tea.Msg {
		return searchTickMsg{gen: gen}
	})
}

func (m *browseModel) setTableFocusFromResults() {
	if m.table != nil {
		m.table.SetFocus(len(m.results) > 0)
	}
}

func (m *browseModel) browseApplyResultRows(results []repo.SearchResult) tea.Cmd {
	m.results = results
	rows := make([][]string, 0, len(results))
	for _, r := range results {
		cat := shared.PlaceholderNA
		if len(r.Categories) > 0 {
			cat = r.Categories[0]
		}
		lic := r.License
		if lic == "" {
			lic = shared.PlaceholderNA
		}
		rows = append(rows, []string{r.Name, r.ID, lic, cat, r.SourceName})
	}
	m.table.SetRows(rows)
	m.table.SetSelectedRow(0)
	m.dismissToastPopup()
	m.setTableFocusFromResults()
	return m.syncTableDimensions()
}

func (m *browseModel) applySearch() tea.Cmd {
	q := strings.TrimSpace(m.searchInput.Value())
	category := m.browseSelectedCategory()
	if q == "" {
		if category == "" && !browseShowFullCatalogInitially {
			m.results = nil
			m.table.SetRows([][]string{})
			m.dismissToastPopup()
			m.setTableFocusFromResults()
			return m.syncTableDimensions()
		}
		results, err := m.repository.ListCatalogFonts(category, browseCatalogUsesSearchSortSettings)
		if err != nil {
			m.showToastPopup(ui.RenderError(err.Error()))
			m.results = nil
			m.table.SetRows([][]string{})
			m.setTableFocusFromResults()
			return m.syncTableDimensions()
		}
		return m.browseApplyResultRows(results)
	}

	results, err := m.repository.SearchFonts(q, category)
	if err != nil {
		m.showToastPopup(ui.RenderError(err.Error()))
		m.results = nil
		m.table.SetRows([][]string{})
		m.setTableFocusFromResults()
		return m.syncTableDimensions()
	}

	return m.browseApplyResultRows(results)
}

func (m *browseModel) browseSelectedCategory() string {
	if m.browseFilterSlot <= 0 || m.browseFilterSlot > len(m.browseCategoryNames) {
		return ""
	}
	return m.browseCategoryNames[m.browseFilterSlot-1]
}

func (m *browseModel) browseFilterDisplayValue() string {
	if m.browseFilterSlot <= 0 || m.browseFilterSlot > len(m.browseCategoryNames) {
		return "All"
	}
	return m.browseCategoryNames[m.browseFilterSlot-1]
}

func (m *browseModel) cycleBrowseFilter(delta int) {
	n := 1 + len(m.browseCategoryNames)
	if n < 1 {
		n = 1
	}
	m.browseFilterSlot = ((m.browseFilterSlot+delta)%n + n) % n
}

func (m *browseModel) lookupFontInfo(fontID string) (repo.FontInfo, bool) {
	manifest, err := m.repository.GetManifest()
	if err != nil {
		return repo.FontInfo{}, false
	}
	for _, source := range manifest.Sources {
		if f, ok := source.Fonts[fontID]; ok {
			return f, true
		}
	}
	return repo.FontInfo{}, false
}

func browseFontDialogTitle(f repo.FontInfo, fontID string) string {
	src := shared.GetSourceNameFromID(fontID)
	if src == "" {
		return f.Name
	}
	return fmt.Sprintf("%s – %s", f.Name, src)
}

func buildBrowseFontDialogBody(f repo.FontInfo, fontID string) string {
	category := shared.PlaceholderUnknown
	if len(f.Categories) > 0 {
		category = f.Categories[0]
	}
	var lines []string
	add := func(label, val string) {
		if val == "" {
			return
		}
		lines = append(lines, ui.FormLabel.Render(label+":")+" "+ui.Text.Render(val))
	}

	add("ID", fontID)
	add("License", f.License)
	add("Category", category)
	if len(f.Tags) > 0 {
		add("Tags", strings.Join(f.Tags, ", "))
	}
	if f.Popularity > 0 {
		add("Popularity", fmt.Sprintf("%d", f.Popularity))
	}
	if !f.LastModified.IsZero() {
		add("Updated", f.LastModified.Format("02/01/2006 - 15:04"))
	}
	if strings.TrimSpace(f.SourceURL) != "" {
		lines = append(lines, ui.FormLabel.Render("Source:")+" "+ui.FormatTerminalURL(f.SourceURL))
	}
	licenseURL := ""
	if f.LicenseURL != "" {
		licenseURL = f.LicenseURL
	} else if f.SourceURL != "" {
		licenseURL = f.SourceURL
	}
	if licenseURL != "" && strings.TrimSpace(licenseURL) != strings.TrimSpace(f.SourceURL) {
		lines = append(lines, ui.FormLabel.Render("License URL:")+" "+ui.FormatTerminalURL(licenseURL))
	}
	return strings.Join(lines, "\n")
}

func (m *browseModel) openModal() tea.Cmd {
	if len(m.results) == 0 {
		return nil
	}
	idx := m.table.GetSelectedRow()
	if idx < 0 || idx >= len(m.results) {
		return nil
	}
	id := m.results[idx].ID
	font, ok := m.lookupFontInfo(id)
	if !ok {
		m.showToastPopup(ui.RenderError("Could not load font details"))
		return nil
	}
	_, findErr := findFontFilesForRemoval(id, m.fontManager, m.installScope, m.repository)
	installed := findErr == nil
	primary := "Install"
	if installed {
		primary = "Uninstall"
	}
	bg := components.NewButtonGroup([]string{primary, "View online", "Close"}, 0)
	bg.SetFocus(true)
	m.modal = &browseModalState{
		fontID:  id,
		font:    font,
		buttons: bg,
	}
	return nil
}

func (m *browseModel) closeModal() {
	m.modal = nil
}

func browseNormalizeSourceLabel(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return shared.PlaceholderNA
	}
	return s
}

func browseResultFromInstall(fontName, source string, msg installFinishedMsg) (title string, errorTitle bool, body string) {
	fontName = strings.TrimSpace(fontName)
	if fontName == "" {
		fontName = shared.PlaceholderNA
	}
	source = browseNormalizeSourceLabel(source)
	if msg.err != nil {
		return "Error", true, ui.RenderError(msg.err.Error())
	}
	if msg.result == nil {
		return "Error", true, ui.RenderError("Operation failed.")
	}
	switch msg.result.Status {
	case InstallStatusCompleted:
		line := fmt.Sprintf("'%s' successfully installed from %s.", fontName, source)
		return "Installed", false, ui.Text.Render(line)
	case InstallStatusSkipped:
		return "Skipped", false, ui.InfoText.Render(msg.result.Message)
	case InstallStatusFailed:
		errText := msg.result.Message
		if len(msg.result.Errors) > 0 {
			errText = msg.result.Errors[0]
		}
		return "Error", true, ui.RenderError(errText)
	default:
		return "Error", true, ui.InfoText.Render(msg.result.Message)
	}
}

func browseResultFromUninstall(fontName string, installScope platform.InstallationScope, msg uninstallFinishedMsg) (title string, errorTitle bool, body string) {
	fontName = strings.TrimSpace(fontName)
	if fontName == "" {
		fontName = shared.PlaceholderNA
	}
	if msg.err != nil {
		return "Error", true, ui.RenderError(msg.err.Error())
	}
	if msg.result == nil {
		return "Error", true, ui.RenderError("Operation failed.")
	}
	switch msg.result.Status {
	case StatusCompleted:
		var line string
		if installScope == platform.MachineScope {
			line = fmt.Sprintf("'%s' has been uninstalled successfully from this machine.", fontName)
		} else {
			line = fmt.Sprintf("'%s' has been uninstalled successfully.", fontName)
		}
		return "Uninstalled", false, ui.Text.Render(line)
	case StatusSkipped:
		return "Skipped", false, ui.InfoText.Render(msg.result.Message)
	case StatusFailed:
		errText := msg.result.Message
		if len(msg.result.Errors) > 0 {
			errText = msg.result.Errors[0]
		}
		return "Error", true, ui.RenderError(errText)
	default:
		return "Error", true, ui.InfoText.Render(msg.result.Message)
	}
}

func (m *browseModel) openResultModal(title string, errorTitle bool, body string) {
	bg := components.NewButtonGroup([]string{"OK"}, 0)
	bg.SetFocus(true)
	m.resultModal = &browseResultModalState{
		borderTitle: title,
		errorTitle:  errorTitle,
		body:        body,
		buttons:     bg,
	}
}

func (m *browseModel) closeResultModal() {
	m.resultModal = nil
}

func (m *browseModel) handleResultModalUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.resultModal == nil {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.String()
		if k == "esc" || k == "enter" || k == " " {
			m.closeResultModal()
			cmd := m.syncTableDimensions()
			return m, cmd
		}
		action := m.resultModal.buttons.HandleKey(k)
		if action == "ok" {
			m.closeResultModal()
			cmd := m.syncTableDimensions()
			return m, cmd
		}
		return m, nil
	}
	return m, nil
}

func (m *browseModel) waitForOpMsg() tea.Cmd {
	if m.opMsgCh == nil {
		return nil
	}
	ch := m.opMsgCh
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

func (m *browseModel) startInstallByID(fontID, fontName, sourceLabel string) tea.Cmd {
	m.installing = true
	m.statusProgress = 0
	m.statusPhase = "Installing"
	m.installPopupFontName = fontName
	if sourceLabel == "" {
		sourceLabel = shared.PlaceholderNA
	}
	m.installPopupSource = sourceLabel

	fm := m.fontManager
	scope := m.installScope
	force := m.force
	fontDir := m.fontDir

	ch := make(chan tea.Msg, 32)
	m.opMsgCh = ch
	go func() {
		defer close(ch)
		res, err := shared.ResolveFontQuery(fontID)
		if err != nil {
			ch <- installFinishedMsg{err: err, fontID: fontID}
			return
		}
		if res.HasMultipleMatches {
			ch <- installFinishedMsg{err: fmt.Errorf("multiple matches for %q", fontID), fontID: fontID}
			return
		}

		onProgress := func(step string, stepPct float64) {
			ch <- browseOpProgressMsg{phase: step, percent: OverallInstallPercent(0, 1, step, stepPct)}
		}

		ir, ierr := installFont(res.Fonts, res.FontID, fm, scope, force, fontDir, true, onProgress)
		ch <- installFinishedMsg{result: ir, err: ierr, fontID: fontID}
	}()
	return m.waitForOpMsg()
}

func (m *browseModel) startUninstallByID(fontID, fontName, sourceLabel string) tea.Cmd {
	m.removing = true
	m.statusProgress = -1
	m.statusPhase = "Removing"
	m.removingFontName = fontName
	if sourceLabel == "" {
		sourceLabel = shared.PlaceholderNA
	}
	m.removingSourceLabel = sourceLabel

	fm := m.fontManager
	scope := m.installScope
	fontDir := m.fontDir
	repository := m.repository

	ch := make(chan tea.Msg, 8)
	m.opMsgCh = ch
	go func() {
		defer close(ch)
		onProgress := func(step string, stepPct float64) {
			ch <- browseOpProgressMsg{phase: step, percent: OverallRemovePercent(0, 1, step, stepPct)}
		}
		rr, err := removeFont(fontID, fm, scope, fontDir, repository, onProgress)
		ch <- uninstallFinishedMsg{result: rr, err: err, fontID: fontID}
	}()
	return m.waitForOpMsg()
}

func (m *browseModel) handleModalUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.modal == nil {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.String()
		if k == "esc" {
			m.closeModal()
			return m, nil
		}
		action := m.modal.buttons.HandleKey(k)
		switch action {
		case "install":
			id := m.modal.fontID
			name := m.modal.font.Name
			src := shared.GetSourceNameFromID(id)
			m.closeModal()
			return m, m.startInstallByID(id, name, src)
		case "uninstall":
			id := m.modal.fontID
			name := m.modal.font.Name
			src := shared.GetSourceNameFromID(id)
			m.closeModal()
			return m, m.startUninstallByID(id, name, src)
		case "view online":
			url := strings.TrimSpace(m.modal.font.SourceURL)
			if url == "" {
				m.showToastPopup(ui.RenderError("No source URL for this font"))
				return m, nil
			}
			if err := platform.OpenURL(url); err != nil {
				m.showToastPopup(ui.RenderError(err.Error()))
				return m, nil
			}
			return m, nil
		case "close":
			m.closeModal()
			return m, nil
		}
		return m, nil
	}
	return m, nil
}

// contentInnerWidth is the usable width inside the window side margins (see browseWindowHMargin).
func (m *browseModel) contentInnerWidth() int {
	if m.width <= 0 {
		return 0
	}
	if m.width <= 2*browseWindowHMargin {
		return max(1, m.width)
	}
	return m.width - 2*browseWindowHMargin
}

func (m *browseModel) searchCardOuterWidth() int {
	inner := m.contentInnerWidth()
	if inner <= 0 {
		return browseSearchCardOuterMin
	}
	w := browseSearchCardOuterWidth
	if w > inner {
		w = inner
	}
	if w < browseSearchCardOuterMin {
		if inner >= browseSearchCardOuterMin {
			w = browseSearchCardOuterMin
		} else {
			w = inner
		}
	}
	return w
}

func (m *browseModel) syncSearchInputWidth() {
	outer := m.searchCardOuterWidth()
	inner := outer - 2 // between │ and │
	if inner < 8 {
		inner = 8
	}
	// browseSearchInnerLeftPad spaces inside the row before the prompt/text.
	pad := browseSearchInnerLeftPad
	m.searchInput.Width = max(6, inner-pad)
}

func (m *browseModel) searchCardView() string {
	outer := m.searchCardOuterWidth()
	return strings.Join([]string{
		browseSearchTopRule(outer),
		browseSearchMiddleLine(outer, m.searchInput.View()),
		browseSearchBottomRule(outer),
	}, "\n")
}

// searchAndFilterRow places the search card and "Filter: …" side by side (or stacked when narrow).
func (m *browseModel) searchAndFilterRow() string {
	const (
		gapSpaces = 4
		filterMin = 12
	)
	card := m.searchCardView()
	inner := m.contentInnerWidth()
	cardW := lipgloss.Width(card)
	gap := strings.Repeat(" ", gapSpaces)
	gapW := lipgloss.Width(gap)
	if inner < cardW+gapW+filterMin {
		return card + "\n" + m.browseFilterLineView(inner)
	}
	avail := inner - cardW - gapW
	if avail < filterMin {
		avail = filterMin
	}
	filter := m.browseFilterBlockView(avail)
	return lipgloss.JoinHorizontal(lipgloss.Center, card, gap, filter)
}

func (m *browseModel) browseFilterLineView(maxW int) string {
	return ui.Text.Render(m.browseFilterStyledText(maxW))
}

func (m *browseModel) browseFilterBlockView(maxW int) string {
	line := ui.Text.Render(m.browseFilterStyledText(maxW))
	return lipgloss.NewStyle().Height(3).MaxWidth(maxW).AlignVertical(lipgloss.Center).Render(line)
}

func (m *browseModel) browseFilterStyledText(maxW int) string {
	prefix := "Filter: "
	val := m.browseFilterDisplayValue()
	if maxW < 8 {
		maxW = 8
	}
	room := maxW - lipgloss.Width(prefix)
	if room < 3 {
		room = 3
	}
	vis := val
	if lipgloss.Width(val) > room {
		vis = ansi.Truncate(val, room, "…")
	}
	return prefix + ui.TextBold.Render(vis)
}

// browseSearchTopRule renders: ╭─  Search  ───────────╮
func browseSearchTopRule(outer int) string {
	const prefix = "╭─  "
	const gapAfterTitle = "  "
	suffix := "╮"
	title := ui.TextBold.Render("Search")
	fillCount := outer - lipgloss.Width(prefix) - lipgloss.Width(title) - lipgloss.Width(gapAfterTitle) - lipgloss.Width(suffix)
	if fillCount < 0 {
		fillCount = 0
	}
	fill := strings.Repeat("─", fillCount)
	return prefix + title + gapAfterTitle + fill + suffix
}

// browseSearchMiddleLine renders: │ …textinput… │ with no extra blank lines.
func browseSearchMiddleLine(outer int, fieldView string) string {
	inner := outer - 2
	if inner < 4 {
		inner = 4
	}
	avail := inner - browseSearchInnerLeftPad
	if avail < 2 {
		avail = 2
	}
	body := lipgloss.NewStyle().Width(avail).MaxWidth(avail).Align(lipgloss.Left).Render(fieldView)
	line := strings.Repeat(" ", browseSearchInnerLeftPad) + body
	if lipgloss.Width(line) > inner {
		line = ansi.Truncate(line, inner, "")
	}
	for lipgloss.Width(line) < inner {
		line += " "
	}
	return "│" + line + "│"
}

// browseSearchBottomRule renders: ╰──────────────────╯
func browseSearchBottomRule(outer int) string {
	if outer < 2 {
		return "╰╯"
	}
	return "╰" + strings.Repeat("─", outer-2) + "╯"
}

// browseApplyHorizontalMargins adds left and right padding to each line so content stays
// within terminalWidth (without reflowing ANSI/borders like lipgloss.Width on the whole view).
func browseApplyHorizontalMargins(block string, terminalWidth, margin int) string {
	if margin <= 0 || terminalWidth <= 0 {
		return block
	}
	inner := terminalWidth - 2*margin
	if inner < 1 {
		inner = 1
	}
	lines := strings.Split(block, "\n")
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		line = ansi.Truncate(line, inner, "")
		w := lipgloss.Width(line)
		pad := inner - w
		if pad < 0 {
			pad = 0
		}
		b.WriteString(strings.Repeat(" ", margin))
		b.WriteString(line)
		b.WriteString(strings.Repeat(" ", pad))
		b.WriteString(strings.Repeat(" ", margin))
	}
	return b.String()
}

func (m *browseModel) helpLineView() string {
	left := ui.Text.Render(strings.Join([]string{
		ui.RenderKeyWithDescription("↑/↓", "Results"),
		ui.RenderKeyWithDescription("Enter", "Details"),
		ui.RenderKeyWithDescription("Esc", "Clear / Quit"),
		ui.RenderKeyWithDescription("Tab", "Filter"),
		ui.RenderKeyWithDescription("Shift+Tab", "Filter back"),
	}, "  "))
	right := ui.SecondaryText.Render(fmt.Sprintf("%d/%d", len(m.results), m.browseManifestFontCount))
	inner := m.contentInnerWidth()
	if inner < 1 {
		inner = 1
	}
	rw := lipgloss.Width(right)
	if rw >= inner {
		return "\n" + ansi.Truncate(right, inner, "")
	}
	lw := lipgloss.Width(left)
	spaces := inner - lw - rw
	if spaces < 1 {
		maxLeft := inner - rw - 1
		if maxLeft < 1 {
			maxLeft = 1
		}
		left = ansi.Truncate(left, maxLeft, "…")
		lw = lipgloss.Width(left)
		spaces = inner - lw - rw
		if spaces < 1 {
			spaces = 1
		}
	}
	line := left + strings.Repeat(" ", spaces) + right
	if lipgloss.Width(line) > inner {
		line = ansi.Truncate(line, inner, "")
	}
	return "\n" + line
}

// browseHeightTitleAndSearchPrefix matches baseView through the spacer before the table:
// title + browseApplyHorizontalMargins(searchAndFilter+"\n\n", …).
// Using margined(search)+"\\n\\n" here underestimates/overestimates vs real layout and
// wastes terminal rows under the help bar.
func (m *browseModel) browseHeightTitleAndSearchPrefix() int {
	title := ui.PageTitle.Render(browsePageTitle) + "\n\n"
	prefixBody := m.searchAndFilterRow() + "\n\n"
	margined := browseApplyHorizontalMargins(prefixBody, m.width, browseWindowHMargin)
	return lipgloss.Height(title + margined)
}

func (m *browseModel) browseHeightHelpSection() int {
	return lipgloss.Height(browseApplyHorizontalMargins(m.helpLineView(), m.width, browseWindowHMargin))
}

func (m *browseModel) calcTableHeight() int {
	above := m.browseHeightTitleAndSearchPrefix()
	help := m.browseHeightHelpSection()
	// lipgloss.Height counts "\n". When we measure vertical slices separately
	// (title+search, table, help) we effectively double-count the seam lines
	// versus the final merged frame, so give the table back those rows.
	avail := m.height - above - help + 2
	if avail < 4 {
		avail = 4
	}
	if avail > m.height {
		avail = m.height
	}
	return avail
}

func (m *browseModel) syncTableDimensions() tea.Cmd {
	if m.table == nil || m.width == 0 || m.height == 0 {
		return nil
	}
	ws := tea.WindowSizeMsg{Width: m.contentInnerWidth(), Height: m.height}

	apply := func(totalLines int) tea.Cmd {
		if totalLines < 4 {
			totalLines = 4
		}
		if totalLines > m.height {
			totalLines = m.height
		}
		updated, cmd := m.table.UpdateWithHeight(ws, totalLines)
		if updated != nil {
			m.table = updated
		}
		return cmd
	}

	target := m.height
	baseH := func() int { return lipgloss.Height(m.baseView()) }

	startEst := m.calcTableHeight()
	startEst = max(4, min(startEst, m.height))

	best := 4
	var cmd tea.Cmd
	for a := startEst; a <= m.height; a++ {
		cmd = apply(a)
		h := baseH()
		if h <= target {
			best = a
		}
		if h >= target {
			break
		}
	}
	for baseH() > target && best > 4 {
		best--
		cmd = apply(best)
	}

	m.tableViewportH = best
	return cmd
}

func (m *browseModel) toastOverlayView() string {
	if m.toastPopup == nil {
		return ""
	}
	footer := ui.Text.Render("Enter or Esc — dismiss")
	boxW := min(56, m.width-4)
	if boxW < 24 {
		boxW = 24
	}
	inner := m.toastPopup.body + "\n\n" + footer
	return ui.CardBorder.Width(boxW).Padding(1, 2).Render(inner)
}

func (m *browseModel) routesTableNav(k string) bool {
	// Only arrow keys — j/k would steal letters from the search query.
	return k == "up" || k == "down"
}

func (m *browseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.syncSearchInputWidth()
		cmd := m.syncTableDimensions()
		return m, cmd

	case searchTickMsg:
		if msg.gen != m.debounceGen || m.installing || m.removing || m.modal != nil || m.resultModal != nil || m.toastPopup != nil {
			return m, nil
		}
		cmd := m.applySearch()
		return m, cmd

	case browseOpProgressMsg:
		if !m.installing && !m.removing {
			return m, nil
		}
		if strings.TrimSpace(msg.phase) != "" {
			m.statusPhase = msg.phase
		}
		m.statusProgress = msg.percent
		return m, m.waitForOpMsg()

	case uninstallFinishedMsg:
		fontName := m.removingFontName
		m.removing = false
		m.removingFontName = ""
		m.removingSourceLabel = ""
		m.statusProgress = 0
		m.statusPhase = ""
		m.opMsgCh = nil
		title, errTitle, body := browseResultFromUninstall(fontName, m.installScope, msg)
		m.openResultModal(title, errTitle, body)
		cmd := m.syncTableDimensions()
		return m, cmd

	case installFinishedMsg:
		fontName := m.installPopupFontName
		source := m.installPopupSource
		m.installing = false
		m.installPopupFontName = ""
		m.installPopupSource = ""
		m.statusProgress = 0
		m.statusPhase = ""
		m.opMsgCh = nil
		title, errTitle, body := browseResultFromInstall(fontName, source, msg)
		m.openResultModal(title, errTitle, body)
		cmd := m.syncTableDimensions()
		return m, cmd
	}

	if m.installing || m.removing {
		if km, ok := msg.(tea.KeyMsg); ok && km.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m, nil
	}

	if m.resultModal != nil {
		return m.handleResultModalUpdate(msg)
	}

	if m.modal != nil {
		return m.handleModalUpdate(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.String()

		if m.toastPopup != nil {
			if k == "enter" || k == "esc" {
				m.dismissToastPopup()
				cmd := m.syncTableDimensions()
				return m, cmd
			}
			return m, nil
		}

		if k == "ctrl+c" {
			return m, tea.Quit
		}

		if k == "esc" {
			if strings.TrimSpace(m.searchInput.Value()) != "" {
				m.searchInput.SetValue("")
				cmd := m.applySearch()
				return m, cmd
			}
			return m, tea.Quit
		}

		if k == "tab" {
			m.cycleBrowseFilter(1)
			cmd := m.applySearch()
			return m, tea.Batch(cmd, textinput.Blink)
		}
		if k == "shift+tab" {
			m.cycleBrowseFilter(-1)
			cmd := m.applySearch()
			return m, tea.Batch(cmd, textinput.Blink)
		}

		if len(m.results) > 0 && m.routesTableNav(k) {
			if m.table != nil {
				updated, tcmd := m.table.Update(msg)
				if updated != nil {
					m.table = updated
				}
				return m, tcmd
			}
			return m, nil
		}

		if len(m.results) > 0 && k == "enter" {
			return m, m.openModal()
		}

		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		next := tea.Batch(cmd, m.scheduleSearch())
		return m, next
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

func (m *browseModel) tableAreaView() string {
	if m.table == nil {
		return ""
	}
	return m.table.View()
}

func (m *browseModel) baseView() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	title := ui.PageTitle.Render(browsePageTitle) + "\n\n"
	body := m.searchAndFilterRow() + "\n\n" + m.tableAreaView() + m.helpLineView()
	margined := browseApplyHorizontalMargins(body, m.width, browseWindowHMargin)

	// Do not apply Width() to the full layout: wrapping reflow breaks rounded
	// border runes and can leave stray bracket glyphs next to the search card.
	return title + margined
}

func (m *browseModel) modalView() string {
	if m.modal == nil {
		return ""
	}
	title := browseFontDialogTitle(m.modal.font, m.modal.fontID)
	body := buildBrowseFontDialogBody(m.modal.font, m.modal.fontID)
	maxW := m.contentInnerWidth()
	if maxW <= 0 {
		maxW = components.DefaultDialogMaxWidth
	}
	if maxW > components.DefaultDialogMaxWidth {
		maxW = components.DefaultDialogMaxWidth
	}
	return components.RenderDialog(title, body, m.modal.buttons, components.DialogOpts{
		MaxWidth: maxW,
		MinWidth: 44,
	})
}

func (m *browseModel) resultModalView() string {
	if m.resultModal == nil {
		return ""
	}
	maxW := m.contentInnerWidth()
	if maxW <= 0 {
		maxW = components.DefaultDialogMaxWidth
	}
	if maxW > components.DefaultDialogMaxWidth {
		maxW = components.DefaultDialogMaxWidth
	}
	return components.RenderDialog(m.resultModal.borderTitle, m.resultModal.body, m.resultModal.buttons, components.DialogOpts{
		MaxWidth:   maxW,
		MinWidth:   44,
		ErrorTitle: m.resultModal.errorTitle,
	})
}

func (m *browseModel) View() string {
	view := m.baseView()
	if m.installing {
		maxOuter := m.contentInnerWidth()
		if maxOuter <= 0 {
			maxOuter = components.DefaultStatusPopupMaxOuter
		}
		if maxOuter > components.DefaultStatusPopupMaxOuter {
			maxOuter = components.DefaultStatusPopupMaxOuter
		}
		phase := strings.TrimSpace(m.statusPhase)
		if phase == "" {
			phase = "Installing"
		}
		popup := components.RenderStatusPopup(phase, m.installPopupFontName, m.installPopupSource, m.statusProgress, maxOuter)
		view = components.Composite(popup, view, components.Center, components.Center, 0, 0)
	}
	if m.removing {
		maxOuter := m.contentInnerWidth()
		if maxOuter <= 0 {
			maxOuter = components.DefaultStatusPopupMaxOuter
		}
		if maxOuter > components.DefaultStatusPopupMaxOuter {
			maxOuter = components.DefaultStatusPopupMaxOuter
		}
		phase := strings.TrimSpace(m.statusPhase)
		if phase == "" {
			phase = "Removing"
		}
		mid := fmt.Sprintf("'%s' from '%s'", m.removingFontName, m.removingSourceLabel)
		popup := components.RenderStatusPopupPlain(phase, mid, m.statusProgress, maxOuter)
		view = components.Composite(popup, view, components.Center, components.Center, 0, 0)
	}
	if m.toastPopup != nil {
		tg := m.toastOverlayView()
		if tg != "" {
			view = components.Composite(tg, view, components.Center, components.Center, 0, 0)
		}
	}
	if m.modal != nil {
		fg := m.modalView()
		if fg != "" {
			view = components.Composite(fg, view, components.Center, components.Center, 0, 0)
		}
	}
	if m.resultModal != nil {
		rg := m.resultModalView()
		if rg != "" {
			view = components.Composite(rg, view, components.Center, components.Center, 0, 0)
		}
	}
	if m.width > 0 && m.height > 0 {
		return ui.FillTerminalArea(view, m.width, m.height)
	}
	return view
}

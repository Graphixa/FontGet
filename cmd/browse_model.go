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
)

type searchTickMsg struct {
	gen int
}

type installFinishedMsg struct {
	err    error
	result *InstallResult
	fontID string
}

type browseModalState struct {
	fontID  string
	font    repo.FontInfo
	cards   string
	buttons *components.ButtonGroup
}

// browseToastPopup is a dismissible status message (Enter / Esc).
type browseToastPopup struct {
	body string
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

	modal *browseModalState

	toastPopup *browseToastPopup

	installing  bool
	debounceGen int

	tableViewportH int
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
			{Header: "Name", Truncatable: true, MinWidth: 6},
			{Header: "Font ID", Truncatable: false, MinWidth: 8},
			{Header: "License", MaxWidth: 18, Truncatable: true, MinWidth: 6},
			{Header: "Categories", Truncatable: true, MinWidth: 6},
			{Header: "Source", Truncatable: false, MinWidth: 6},
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
		searchInput:    ti,
		table:          tm,
		repository:     repository,
		fontManager:    fontManager,
		installScope:   installScope,
		fontDir:        fontDir,
		force:          force,
		tableViewportH: 8,
	}, nil
}

func (m *browseModel) Init() tea.Cmd {
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

func (m *browseModel) applySearch() tea.Cmd {
	q := strings.TrimSpace(m.searchInput.Value())
	if q == "" {
		m.results = nil
		m.table.SetRows([][]string{})
		m.dismissToastPopup()
		m.setTableFocusFromResults()
		return m.syncTableDimensions()
	}

	results, err := m.repository.SearchFonts(q, "")
	if err != nil {
		m.showToastPopup(ui.RenderError(err.Error()))
		m.results = nil
		m.table.SetRows([][]string{})
		m.setTableFocusFromResults()
		return m.syncTableDimensions()
	}

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

func (m *browseModel) buildInfoCards(font repo.FontInfo, fontID string) string {
	category := InfoPlaceholderUnknown
	if len(font.Categories) > 0 {
		category = font.Categories[0]
	}
	tags := ""
	if len(font.Tags) > 0 {
		tags = strings.Join(font.Tags, ", ")
	}
	lastModified := ""
	if !font.LastModified.IsZero() {
		lastModified = font.LastModified.Format("02/01/2006 - 15:04")
	}
	popularity := ""
	if font.Popularity > 0 {
		popularity = fmt.Sprintf("%d", font.Popularity)
	}

	cards := []components.Card{
		components.FontDetailsCard(font.Name, fontID, category, tags, lastModified, font.SourceURL, popularity),
	}
	licenseURL := ""
	if font.LicenseURL != "" {
		licenseURL = font.LicenseURL
	} else if font.SourceURL != "" {
		licenseURL = font.SourceURL
	}
	cards = append(cards, components.LicenseInfoCard(font.License, licenseURL))

	cm := components.NewCardModel("", cards)
	w := m.width - 4
	if w > 120 {
		w = 120
	}
	if w < 40 {
		w = 40
	}
	cm.SetWidth(w - 2)
	return cm.Render()
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
	bg := components.NewButtonGroup([]string{"Install", "View online", "Close"}, 0)
	bg.SetFocus(true)
	m.modal = &browseModalState{
		fontID:  id,
		font:    font,
		cards:   m.buildInfoCards(font, id),
		buttons: bg,
	}
	return nil
}

func (m *browseModel) closeModal() {
	m.modal = nil
}

func (m *browseModel) startInstallByID(fontID string) tea.Cmd {
	m.installing = true

	fm := m.fontManager
	scope := m.installScope
	force := m.force
	fontDir := m.fontDir

	return func() tea.Msg {
		res, err := shared.ResolveFontQuery(fontID)
		if err != nil {
			return installFinishedMsg{err: err, fontID: fontID}
		}
		if res.HasMultipleMatches {
			return installFinishedMsg{
				err:    fmt.Errorf("multiple matches for %q", fontID),
				fontID: fontID,
			}
		}
		ir, ierr := installFont(res.Fonts, res.FontID, fm, scope, force, fontDir)
		return installFinishedMsg{result: ir, err: ierr, fontID: fontID}
	}
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
			m.closeModal()
			return m, m.startInstallByID(id)
		case "view online":
			url := strings.TrimSpace(m.modal.font.SourceURL)
			m.closeModal()
			if url == "" {
				m.showToastPopup(ui.RenderError("No source URL for this font"))
				return m, nil
			}
			if err := platform.OpenURL(url); err != nil {
				m.showToastPopup(ui.RenderError(err.Error()))
				return m, nil
			}
			m.showToastPopup(ui.InfoText.Render("Opened in browser"))
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
	parts := []string{
		ui.RenderKeyWithDescription("↑/↓", "Results"),
		ui.RenderKeyWithDescription("Enter", "Details"),
		ui.RenderKeyWithDescription("Esc", "Clear · Quit"),
	}
	return "\n" + ui.Text.Render(strings.Join(parts, "  "))
}

func (m *browseModel) topSectionHeight() int {
	title := ui.PageTitle.Render(browsePageTitle) + "\n\n"
	search := browseApplyHorizontalMargins(m.searchCardView(), m.width, browseWindowHMargin)
	return lipgloss.Height(title + search + "\n\n")
}

func (m *browseModel) calcTableHeight() int {
	topH := m.topSectionHeight()
	helpH := lipgloss.Height(browseApplyHorizontalMargins(m.helpLineView(), m.width, browseWindowHMargin))
	pad := 2
	avail := m.height - topH - helpH - pad
	if avail < 4 {
		avail = 4
	}
	return avail
}

func (m *browseModel) syncTableDimensions() tea.Cmd {
	if m.table == nil || m.width == 0 || m.height == 0 {
		return nil
	}
	m.tableViewportH = m.calcTableHeight()
	ws := tea.WindowSizeMsg{Width: m.contentInnerWidth(), Height: m.height}
	updated, cmd := m.table.UpdateWithHeight(ws, m.tableViewportH)
	if updated != nil {
		m.table = updated
	}
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

func (m *browseModel) installingBannerView() string {
	return ui.InfoText.Render("Installing…")
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
		if msg.gen != m.debounceGen || m.installing || m.modal != nil || m.toastPopup != nil {
			return m, nil
		}
		cmd := m.applySearch()
		return m, cmd

	case installFinishedMsg:
		m.installing = false
		if msg.err != nil {
			m.showToastPopup(ui.RenderError(msg.err.Error()))
			cmd := m.syncTableDimensions()
			return m, cmd
		}
		if msg.result != nil {
			switch msg.result.Status {
			case InstallStatusCompleted:
				m.showToastPopup(ui.SuccessText.Render(msg.result.Message))
			case InstallStatusSkipped:
				m.showToastPopup(ui.InfoText.Render(msg.result.Message))
			case InstallStatusFailed:
				errText := msg.result.Message
				if len(msg.result.Errors) > 0 {
					errText = msg.result.Errors[0]
				}
				m.showToastPopup(ui.RenderError(errText))
			default:
				m.showToastPopup(ui.InfoText.Render(msg.result.Message))
			}
		}
		cmd := m.syncTableDimensions()
		return m, cmd
	}

	if m.installing {
		if km, ok := msg.(tea.KeyMsg); ok && km.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m, nil
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
	body := m.searchCardView() + "\n\n" + m.tableAreaView() + m.helpLineView()
	margined := browseApplyHorizontalMargins(body, m.width, browseWindowHMargin)

	// Do not apply Width() to the full layout: wrapping reflow breaks rounded
	// border runes and can leave stray bracket glyphs next to the search card.
	return title + margined
}

func (m *browseModel) modalView() string {
	if m.modal == nil {
		return ""
	}
	content := m.modal.cards + "\n\n" + m.modal.buttons.Render()
	return ui.CardBorder.Padding(1, 2).Render(content)
}

func (m *browseModel) View() string {
	view := m.baseView()
	if m.installing {
		banner := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(m.installingBannerView())
		view = components.Composite(banner, view, components.Center, components.Bottom, 0, -1)
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
	return view
}

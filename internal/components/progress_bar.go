package components

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"fontget/internal/shared"
	"fontget/internal/ui"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
)

// OperationItem represents a single item in the progress display
type OperationItem struct {
	Name          string   // "Roboto"
	SourceName    string   // "Google Fonts", "Font Squirrel", etc.
	Status        string   // "completed", "in_progress", "pending", "failed"
	StatusMessage string   // "Installed", "Downloading from X", etc.
	ErrorMessage  string   // Brief error message for failed items (shown in error color)
	Variants      []string // For verbose mode: ["Roboto-Regular.ttf", ...]
	Scope         string   // For remove command: "user scope"
}

// ProgressBarModel manages the progress display
type ProgressBarModel struct {
	Title         string
	TotalItems    int
	Items         []OperationItem
	VerboseMode   bool // Show operational details and file/variant listings (static display, no spinner)
	DebugMode     bool // Show technical details (static display, no spinner)
	ProgressBar   progress.Model
	Spinner       spinner.Model
	operationFunc func(program *tea.Program) error
	quitting      bool
	cancelled     bool // Track if operation was cancelled
	err           error
	program       *tea.Program
	statusReport *StatusReportData
	cancelChan   chan struct{} // Channel to signal cancellation
	opDone       chan struct{}
	width        int // terminal columns; 0 until known
	height       int // terminal rows; 0 until known
}

// Message types for communication
type ItemUpdateMsg struct {
	Index        int
	Name         string // Optional: update the item name
	Status       string
	Message      string
	ErrorMessage string // Brief error message for failed items
	Variants     []string
	Scope        string
	SourceName   string
}

type ProgressUpdateMsg struct {
	Percent float64
}

// TitleUpdateMsg updates the progress bar title dynamically
type TitleUpdateMsg struct {
	Title string
}

// TotalItemsUpdateMsg updates the total items count dynamically
type TotalItemsUpdateMsg struct {
	TotalItems int
}

type operationCompleteMsg struct {
	err error
}

type quitMsg struct{}

// StatusReportData represents status report data for the progress display
type StatusReportData struct {
	Success      int
	Skipped      int
	Failed       int
	SuccessLabel string
	SkippedLabel string
	FailedLabel  string
}

type StatusReportMsg struct {
	Report StatusReportData
}

// operationTickMsg is sent to update the progress display
type operationTickMsg time.Time

// NewProgressBar creates a new progress bar model
func NewProgressBar(title string, items []OperationItem, verboseMode bool, debugMode bool) *ProgressBarModel {
	// Create progress bar with gradient colors
	startColor, endColor := ui.GetProgressBarGradient()
	prog := progress.New(
		progress.WithGradient(startColor, endColor),
	)
	prog.Width = 30 // Match design width

	// Create spinner (no style - we'll apply it ourselves when rendering)
	spin := spinner.New()
	spin.Spinner = spinner.Dot

	return &ProgressBarModel{
		Title:       title,
		TotalItems:  len(items),
		Items:       items,
		VerboseMode: verboseMode,
		DebugMode:   debugMode,
		ProgressBar: prog,
		Spinner:     spin,
		cancelChan: make(chan struct{}),
		opDone:     make(chan struct{}),
	}
}

func (m ProgressBarModel) Init() tea.Cmd {
	return tea.Batch(
		operationTickCmd(),
		m.Spinner.Tick,
		m.startOperation(),
	)
}

// startOperation runs the actual work in background
func (m ProgressBarModel) startOperation() tea.Cmd {
	return func() tea.Msg {
		go func() {
			defer func() {
				if m.opDone != nil {
					select {
					case <-m.opDone:
					default:
						close(m.opDone)
					}
				}
			}()
			err := m.operationFunc(m.program)
			select {
			case <-m.cancelChan:
				err = shared.ErrOperationCancelled
			default:
			}
			if m.program != nil {
				m.program.Send(operationCompleteMsg{err: err})
			}
		}()
		return nil
	}
}

func (m ProgressBarModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		// Handle cancellation keys (q, esc, ctrl+c, enter, space) - like sources update
		switch key {
		case "q", "ctrl+c", "esc", "enter", " ":
			if m.cancelled {
				return m, tea.Quit
			}
			m.quitting = true
			m.cancelled = true
			m.err = shared.ErrOperationCancelled
			m.Title = "Cancelling..."
			if m.cancelChan != nil {
				select {
				case <-m.cancelChan:
				default:
					close(m.cancelChan)
				}
			}
			// Quit TUI immediately; RunProgressBar joins the worker after p.Run().
			return m, tea.Quit
		}
		// If operation is complete, any other key still joins then quits.
		if m.quitting {
			return m, waitForOperationQuit(m.opDone)
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ProgressBar.Width = msg.Width - 8
		if m.ProgressBar.Width > 80 {
			m.ProgressBar.Width = 80
		}
		return m, nil

	case ItemUpdateMsg:
		if msg.Index >= 0 && msg.Index < len(m.Items) {
			if msg.Name != "" {
				m.Items[msg.Index].Name = msg.Name
			}
			m.Items[msg.Index].Status = msg.Status
			if msg.Message != "" {
				m.Items[msg.Index].StatusMessage = msg.Message
			}
			if msg.ErrorMessage != "" {
				m.Items[msg.Index].ErrorMessage = msg.ErrorMessage
			}
			if msg.Variants != nil {
				m.Items[msg.Index].Variants = msg.Variants
			}
			if msg.Scope != "" {
				m.Items[msg.Index].Scope = msg.Scope
			}
			if msg.SourceName != "" {
				m.Items[msg.Index].SourceName = msg.SourceName
			}
		}
		return m, nil

	case ProgressUpdateMsg:
		// Don't process progress updates after operation completes
		if m.quitting {
			return m, nil
		}
		// Update progress bar
		cmd := m.ProgressBar.SetPercent(msg.Percent / 100.0)
		return m, cmd

	case TitleUpdateMsg:
		// Update the title dynamically
		m.Title = msg.Title
		return m, nil

	case TotalItemsUpdateMsg:
		// Update the total items count dynamically
		m.TotalItems = msg.TotalItems
		// If we're adding items, we may need to expand the Items slice
		if msg.TotalItems > len(m.Items) {
			// Expand items slice with pending items
			for len(m.Items) < msg.TotalItems {
				m.Items = append(m.Items, OperationItem{
					Name:   "",
					Status: "pending",
				})
			}
		}
		return m, nil

	case StatusReportMsg:
		m.statusReport = &msg.Report
		return m, nil

	case operationCompleteMsg:
		// Ignore completion messages if we've already quit (interrupted)
		if m.quitting {
			return m, nil
		}
		m.quitting = true
		m.err = msg.err
		// Reach 100% only on successful completion — never fabricate progress on failure/cancel.
		var cmd tea.Cmd
		if msg.err == nil {
			cmd = m.ProgressBar.SetPercent(1.0)
		}
		// For progress bars without items, quit immediately (no delay needed)
		// For progress bars with items, show final state briefly before quitting
		if m.TotalItems == 0 {
			// No items to show - set progress and quit after processing one frame
			// The frame message will update the progress bar, then we'll quit
			return m, cmd
		}
		// Show final state with items, then quit after a brief delay
		// Reduced from 2s to 300ms for better responsiveness
		if cmd != nil {
			return m, tea.Batch(
				cmd,
				tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg {
					return quitMsg{}
				}),
			)
		}
		return m, tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg {
			return quitMsg{}
		})

	case quitMsg:
		// Handle explicit quit message
		return m, tea.Quit

	case operationTickMsg:
		if m.quitting {
			// Stop tick timer when quitting
			return m, nil
		}
		// Keep the progress bar animating by continuously updating it
		// This is what makes the progress bar animate smoothly
		cmd := m.ProgressBar.SetPercent(m.ProgressBar.Percent())
		// Schedule next tick and process progress bar update
		return m, tea.Batch(operationTickCmd(), cmd)

	case progress.FrameMsg:
		// Handle progress bar animation
		progressModel, cmd := m.ProgressBar.Update(msg)
		m.ProgressBar = progressModel.(progress.Model)
		// If we're quitting and have no items, quit after processing this frame
		if m.quitting && m.TotalItems == 0 {
			return m, tea.Quit
		}
		return m, cmd

	case spinner.TickMsg:
		// Handle spinner animation
		spinnerModel, cmd := m.Spinner.Update(msg)
		m.Spinner = spinnerModel
		return m, cmd

	default:
		return m, nil
	}
}

// itemBlock is one result row, including verbose variant lines. Trailing newline included.
func (m ProgressBarModel) itemBlock(item OperationItem) string {
	var styledIcon string
	switch item.Status {
	case "completed":
		styledIcon = ui.SuccessText.Render("✓")
	case "failed":
		styledIcon = ui.ErrorText.Render("✗")
	case "skipped":
		styledIcon = ui.SuccessText.Render("✓")
	case "in_progress":
		if m.VerboseMode || m.DebugMode {
			styledIcon = "○"
		} else {
			styledIcon = strings.TrimSpace(m.Spinner.View())
			var spinnerColor lipgloss.TerminalColor
			if ui.SpinnerColor == "" {
				spinnerColor = lipgloss.NoColor{}
			} else {
				spinnerColor = lipgloss.Color(ui.SpinnerColor)
			}
			styledIcon = lipgloss.NewStyle().Foreground(spinnerColor).Render(styledIcon)
		}
	default:
		styledIcon = " "
	}

	var sourcePart string
	if item.SourceName != "" {
		sourcePart = " " + ui.InfoText.Render(fmt.Sprintf("[%s]", item.SourceName))
	}

	var statusText string
	switch item.Status {
	case "completed":
		actionWord := item.StatusMessage
		if actionWord == "" {
			actionWord = "Installed"
		}
		if strings.EqualFold(actionWord, "Removed") {
			if item.Scope != "" {
				statusText = fmt.Sprintf("%s from %s", actionWord, item.Scope)
			} else {
				statusText = actionWord
			}
		} else {
			statusText = actionWord
		}
	case "skipped":
		statusText = "Skipped... already installed"
	case "failed":
		if item.ErrorMessage != "" {
			statusText = ui.ErrorText.Render(item.ErrorMessage)
		} else {
			statusText = "Installation failed"
		}
	case "in_progress":
		if item.StatusMessage != "" {
			statusText = item.StatusMessage
		} else {
			statusText = "Installing..."
		}
	default:
		if item.StatusMessage != "" {
			statusText = item.StatusMessage
		} else {
			statusText = "Pending..."
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "  %s %s%s - %s\n", styledIcon, item.Name, sourcePart, statusText)
	if m.VerboseMode && len(item.Variants) > 0 {
		for _, variant := range item.Variants {
			fmt.Fprintf(&b, "      ↳ %s\n", variant)
		}
	}
	return b.String()
}

// viewportMargin keeps the live View off the last terminal rows so Windows
// does not scroll. Older item rows are clipped (not printed to scrollback).
const viewportMargin = 3

func (m ProgressBarModel) View() string {
	return strings.Join(m.viewLines(), "\n")
}

func (m ProgressBarModel) viewLines() []string {
	idxs := m.clippedItemIndices()
	var lines []string
	if !m.VerboseMode && !m.DebugMode {
		lines = append(lines, m.fitLine(m.headerLine()))
		if len(idxs) > 0 {
			lines = append(lines, "")
		}
	}
	for _, i := range idxs {
		block := strings.TrimRight(m.itemBlock(m.Items[i]), "\n")
		for _, line := range strings.Split(block, "\n") {
			lines = append(lines, m.fitLine(line))
		}
	}
	return lines
}

func (m ProgressBarModel) headerLine() string {
	completed := 0
	for _, item := range m.Items {
		if item.Status == "completed" || item.Status == "failed" || item.Status == "skipped" {
			completed++
		}
	}
	return fmt.Sprintf("%s (%d of %d) %s", m.Title, completed, m.TotalItems, m.renderInlineProgressBar())
}

// clippedItemIndices keeps the newest started rows that fit under the bar.
func (m ProgressBarModel) clippedItemIndices() []int {
	var idxs []int
	for i, item := range m.Items {
		if item.Status == "pending" || item.Name == "" {
			continue
		}
		idxs = append(idxs, i)
	}
	budget := m.itemLineBudget()
	for len(idxs) > 1 && m.itemLines(idxs) > budget {
		idxs = idxs[1:]
	}
	return idxs
}

func (m ProgressBarModel) itemLineBudget() int {
	if m.height <= viewportMargin {
		return 1 << 20
	}
	header := 2 // title + blank under it
	if m.VerboseMode || m.DebugMode {
		header = 0
	}
	room := m.height - viewportMargin - header
	if room < 1 {
		room = 1
	}
	return room
}

func (m ProgressBarModel) itemLines(idxs []int) int {
	n := 0
	for _, i := range idxs {
		block := strings.TrimRight(m.itemBlock(m.Items[i]), "\n")
		if block == "" {
			continue
		}
		n += strings.Count(block, "\n") + 1
	}
	return n
}

func (m ProgressBarModel) fitLine(line string) string {
	if m.width <= 1 {
		return line
	}
	return ansi.Truncate(line, m.width-1, "")
}

// InlineProgressBarView renders the same gradient block + percent label used by the CLI progress UI.
// percent0to100 is clamped to [0, 100]; barCharWidth is the number of █/░ cells inside the brackets.
func InlineProgressBarView(percent0to100 float64, barCharWidth int) string {
	if barCharWidth < 1 {
		barCharWidth = 1
	}
	p := percent0to100
	if p < 0 {
		p = 0
	}
	if p > 100 {
		p = 100
	}
	filled := int(float64(barCharWidth) * (p / 100.0))
	if filled > barCharWidth {
		filled = barCharWidth
	}
	empty := barCharWidth - filled

	startColor, endColor := ui.GetProgressBarGradient()
	var barBuilder strings.Builder
	for i := 0; i < filled; i++ {
		var ratio float64
		if filled > 1 {
			ratio = float64(i) / float64(filled-1)
		}
		if ratio > 1.0 {
			ratio = 1.0
		}
		if ratio < 0.0 {
			ratio = 0.0
		}
		gradientColor := interpolateHexColor(startColor, endColor, ratio)
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(gradientColor))
		barBuilder.WriteString(style.Render("█"))
	}
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086")) // Overlay 0 - gray
	for i := 0; i < empty; i++ {
		barBuilder.WriteString(emptyStyle.Render("░"))
	}
	barVisual := barBuilder.String()
	percentText := fmt.Sprintf("%.0f%%", p)
	return fmt.Sprintf("[%s] %s", barVisual, percentText)
}

// renderInlineProgressBar creates a compact progress bar for inline display
func (m ProgressBarModel) renderInlineProgressBar() string {
	barWidth := 15
	if m.ProgressBar.Width < 30 {
		barWidth = 10
	}
	return InlineProgressBarView(m.ProgressBar.Percent()*100, barWidth)
}

// interpolateHexColor interpolates between two hex colors
func interpolateHexColor(start, end string, ratio float64) string {
	// Parse hex colors to RGB
	startRGB := hexToRGB(start)
	endRGB := hexToRGB(end)

	// Interpolate each component
	r := int(float64(startRGB[0])*(1-ratio) + float64(endRGB[0])*ratio)
	g := int(float64(startRGB[1])*(1-ratio) + float64(endRGB[1])*ratio)
	b := int(float64(startRGB[2])*(1-ratio) + float64(endRGB[2])*ratio)

	// Convert back to hex
	return rgbToHex(r, g, b)
}

// hexToRGB converts a hex color string to RGB values
func hexToRGB(hex string) [3]int {
	// Remove # if present
	hex = strings.TrimPrefix(hex, "#")

	// Parse hex string
	var r, g, b int
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)

	return [3]int{r, g, b}
}

// rgbToHex converts RGB values to a hex color string
func rgbToHex(r, g, b int) string {
	// Clamp values to valid range
	if r < 0 {
		r = 0
	}
	if r > 255 {
		r = 255
	}
	if g < 0 {
		g = 0
	}
	if g > 255 {
		g = 255
	}
	if b < 0 {
		b = 0
	}
	if b > 255 {
		b = 255
	}

	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// operationTickCmd returns a command that sends a tick message after 1 second
func operationTickCmd() tea.Cmd {
	return tea.Tick(time.Second*1, func(t time.Time) tea.Msg {
		return operationTickMsg(t)
	})
}

func waitForOperationQuit(done <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		if done != nil {
			<-done
		}
		return quitMsg{}
	}
}

// UseInteractiveRenderer is true only when both stdin and stdout are terminals.
func UseInteractiveRenderer() bool {
	return term.IsTerminal(os.Stdin.Fd()) && term.IsTerminal(os.Stdout.Fd())
}

// RunProgressBar runs the progress display with the given operation.
// Interactive Bubble Tea is used only when stdin and stdout are terminals and debug is off.
func RunProgressBar(title string, items []OperationItem, verboseMode bool, debugMode bool, operation func(send func(msg tea.Msg), cancelChan <-chan struct{}) error) error {
	if debugMode || !UseInteractiveRenderer() {
		return runPlainProgressBar(items, operation)
	}
	return runInteractiveProgressBar(title, items, verboseMode, debugMode, operation)
}

func runPlainProgressBar(items []OperationItem, operation func(send func(msg tea.Msg), cancelChan <-chan struct{}) error) error {
	cancelChan := make(chan struct{})
	trace := os.Getenv("FONTGET_PROGRESS_TRACE") == "1"
	var lastPct float64 = -1
	send := func(msg tea.Msg) {
		switch update := msg.(type) {
		case ProgressUpdateMsg:
			if trace {
				fmt.Fprintf(os.Stderr, "[progress] %5.1f%%\n", update.Percent)
				if lastPct >= 0 && update.Percent+0.05 < lastPct {
					fmt.Fprintf(os.Stderr, "[progress] RESET %.1f%% → %.1f%%\n", lastPct, update.Percent)
				}
				lastPct = update.Percent
			}
		case ItemUpdateMsg:
			if update.Index < 0 || update.Index >= len(items) {
				return
			}
			name := items[update.Index].Name
			if update.Name != "" {
				name = update.Name
			}
			if trace && update.Message != "" {
				fmt.Fprintf(os.Stderr, "[progress] %s — %s\n", name, update.Message)
			}
			switch update.Status {
			case "failed":
				if update.ErrorMessage != "" {
					fmt.Printf("%s: failed: %s\n", name, update.ErrorMessage)
				} else {
					fmt.Printf("%s: failed\n", name)
				}
			case "completed":
				fmt.Printf("%s: installed\n", name)
			case "skipped":
				fmt.Printf("%s: skipped\n", name)
			}
		}
	}
	return operation(send, cancelChan)
}

func runInteractiveProgressBar(title string, items []OperationItem, verboseMode bool, debugMode bool, operation func(send func(msg tea.Msg), cancelChan <-chan struct{}) error) error {
	model := NewProgressBar(title, items, verboseMode, debugMode)
	if w, h, err := term.GetSize(os.Stdout.Fd()); err == nil {
		model.width = w
		model.height = h
	}
	p := tea.NewProgram(model)
	model.program = p
	var workStarted atomic.Bool
	model.operationFunc = func(program *tea.Program) error {
		workStarted.Store(true)
		return operation(func(msg tea.Msg) {
			select {
			case <-model.cancelChan:
				return
			default:
			}
			program.Send(msg)
		}, model.cancelChan)
	}

	finalModel, err := p.Run()
	joinWorker := func() {
		if !workStarted.Load() || model.opDone == nil {
			return
		}
		select {
		case <-model.cancelChan:
		default:
			close(model.cancelChan)
		}
		<-model.opDone
	}
	if err != nil {
		joinWorker()
		return err
	}
	joinWorker()

	if m, ok := finalModel.(ProgressBarModel); ok {
		if m.cancelled {
			return shared.ErrOperationCancelled
		}
		if m.err != nil {
			return m.err
		}
	}
	return nil
}

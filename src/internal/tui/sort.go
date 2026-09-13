package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nerdexecutive/ne-image-sorter/internal/domain"
	"github.com/nerdexecutive/ne-image-sorter/internal/sorter"
)

// sortStage is the sort screen's position in the scan, confirm, apply cycle.
type sortStage int

const (
	stageScanning sortStage = iota
	stagePreview
	stageConfirm
	stageDone
	stageFailed
)

// planMsg carries a completed scan back to the screen.
type planMsg struct {
	plan sorter.Plan
	err  error
}

// reportMsg carries a completed move run back to the screen.
type reportMsg struct {
	report sorter.Report
	err    error
}

// sortModel previews the policy against the source folder, then applies it.
type sortModel struct {
	svc      *sorter.Service
	cfg      domain.Config
	stage    sortStage
	plan     sorter.Plan
	report   sorter.Report
	cursor   int
	showKeep bool
	width    int
	height   int
	err      string
}

func newSortModel(svc *sorter.Service, cfg domain.Config) sortModel {
	return sortModel{svc: svc, cfg: cfg, stage: stageScanning}
}

// Init starts the scan as soon as the screen opens.
func (m sortModel) Init() tea.Cmd {
	return func() tea.Msg {
		plan, err := m.svc.Plan(m.cfg)
		return planMsg{plan: plan, err: err}
	}
}

// Update advances the scan, confirm, apply cycle.
func (m sortModel) Update(msg tea.Msg) (sortModel, tea.Cmd) {
	switch msg := msg.(type) {
	case planMsg:
		if msg.err != nil {
			m.stage, m.err = stageFailed, msg.err.Error()
			return m, nil
		}
		m.plan, m.stage, m.cursor = msg.plan, stagePreview, 0
		return m, nil

	case reportMsg:
		if msg.err != nil {
			m.stage, m.err = stageFailed, msg.err.Error()
			return m, nil
		}
		m.report, m.stage = msg.report, stageDone
		return m, nil

	case tea.KeyMsg:
		return m.updateKey(msg)
	}
	return m, nil
}

// updateKey handles input for whichever stage is active.
func (m sortModel) updateKey(k tea.KeyMsg) (sortModel, tea.Cmd) {
	key := k.String()

	// esc leaves from every stage, which keeps its meaning constant.
	if key == "esc" {
		return m, switchTo(ScreenMenu)
	}

	switch m.stage {
	case stagePreview:
		switch key {
		case "up", "k":
			m.cursor = clampCursor(m.cursor-1, len(m.visible()))
		case "down", "j":
			m.cursor = clampCursor(m.cursor+1, len(m.visible()))
		case "v":
			m.showKeep = !m.showKeep
			m.cursor = 0
		case "enter":
			if m.plan.MoveCount > 0 {
				m.stage = stageConfirm
			}
		}

	case stageConfirm:
		switch key {
		case "y":
			m.stage = stageScanning
			cfg, svc, plan := m.cfg, m.svc, m.plan
			return m, func() tea.Msg {
				report, err := svc.Apply(cfg, plan)
				return reportMsg{report: report, err: err}
			}
		case "n":
			m.stage = stagePreview
		}

	case stageDone, stageFailed:
		if key == "enter" || key == "q" {
			return m, switchTo(ScreenMenu)
		}
	}
	return m, nil
}

// visible returns the decisions the current filter shows.
func (m sortModel) visible() []sorter.Decision {
	if m.showKeep {
		return m.plan.Decisions
	}
	return m.plan.Moves()
}

// View renders the active stage.
func (m sortModel) View() string {
	switch m.stage {
	case stageScanning:
		return m.frame("Working", StyleDim.Render("  Reading image headers…"),
			HelpBar("esc", "back"))
	case stageConfirm:
		return m.viewConfirm()
	case stageDone:
		return m.viewDone()
	case stageFailed:
		return m.frame("Sort failed", "  "+StyleError.Render(m.err)+"\n\n"+
			"  "+StyleDim.Render("Check the folders in the Folders screen, then try again."),
			HelpBar("enter", "back to menu", "esc", "back"))
	default:
		return m.viewPreview()
	}
}

// frame wraps a body in the standard header and footer.
func (m sortModel) frame(title, body, help string) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(RenderHeader(title, m.cfg.SourceDir, m.width))
	b.WriteString("\n\n")
	b.WriteString(body)
	b.WriteString("\n\n")
	b.WriteString(help)
	b.WriteString("\n")
	return b.String()
}

// viewPreview lists what the policy decided, and why.
func (m sortModel) viewPreview() string {
	w := ContentWidth(m.width)
	rows := m.visible()

	var b strings.Builder
	b.WriteString(fmt.Sprintf("  %s  %s   %s  %s\n",
		StyleWarning.Render(fmt.Sprintf("%d to move", m.plan.MoveCount)),
		StyleDim.Render("·"),
		StyleSuccess.Render(fmt.Sprintf("%d to keep", m.plan.KeepCount)),
		StyleDim.Render(fmt.Sprintf("· %d scanned", len(m.plan.Decisions)))))
	b.WriteString("  " + StyleSeparator.Render(strings.Repeat("─", w-2)) + "\n\n")

	if len(rows) == 0 {
		b.WriteString("  " + StyleSuccess.Render("Nothing to move. Every image matches a keep rule.") + "\n")
		b.WriteString("  " + StyleDim.Render("Press v to show the images that are staying.") + "\n")
		return m.frame("Preview", b.String(), HelpBar("v", "show all", "esc", "back"))
	}

	// Show a window around the cursor so long lists stay navigable.
	start, end := window(m.cursor, len(rows), m.listHeight())
	for i := start; i < end; i++ {
		d := rows[i]
		marker, style := "  ", StyleNormal
		if i == m.cursor {
			marker, style = StyleSelected.Render("▸ "), StyleSelected
		}
		verdict := StyleWarning.Render("move")
		if d.Action == domain.ActionKeep {
			verdict = StyleSuccess.Render("keep")
		}
		b.WriteString(fmt.Sprintf("%s%s  %s  %s\n", marker, verdict,
			style.Render(fmt.Sprintf("%-34s", Truncate(d.Image.Name, 34))),
			StyleDim.Render(d.Image.Resolution())))
	}

	if start > 0 || end < len(rows) {
		b.WriteString("\n  " + StyleDim.Render(fmt.Sprintf("showing %d-%d of %d", start+1, end, len(rows))))
		b.WriteString("\n")
	}

	if len(rows) > 0 {
		b.WriteString("\n  " + StyleSeparator.Render(strings.Repeat("─", w-2)) + "\n")
		b.WriteString("  " + StyleDim.Render("why  ") + StyleNormal.Render(
			Truncate(rows[clampCursor(m.cursor, len(rows))].Reason, w-8)) + "\n")
	}

	filter := "show all"
	if m.showKeep {
		filter = "show moves only"
	}
	return m.frame("Preview", b.String(),
		HelpBarWrap(m.width, "↑/↓", "select", "v", filter, "enter", "sort now", "esc", "back"))
}

// viewConfirm gates the only destructive step behind an explicit key.
func (m sortModel) viewConfirm() string {
	var b strings.Builder
	b.WriteString("  " + StyleWarning.Render(fmt.Sprintf("Move %d image(s)?", m.plan.MoveCount)) + "\n\n")
	b.WriteString("  " + StyleDim.Render("from  ") + StyleNormal.Render(m.cfg.SourceDir) + "\n")
	b.WriteString("  " + StyleDim.Render("into  ") + StyleNormal.Render(m.cfg.DestDir) + "\n\n")
	b.WriteString("  " + StyleDim.Render("Files are never overwritten. A name clash is suffixed instead.") + "\n")
	return m.frame("Confirm", b.String(), HelpBar("y", "move them", "n", "go back", "esc", "cancel"))
}

// viewDone reports the outcome, including any file that could not be moved.
func (m sortModel) viewDone() string {
	var b strings.Builder
	b.WriteString("  " + StyleSuccess.Render(fmt.Sprintf("Moved %d image(s) into", m.report.Moved)) + "\n")
	b.WriteString("  " + StyleNormal.Render(m.cfg.DestDir) + "\n")

	if m.report.Failed > 0 {
		b.WriteString("\n  " + StyleError.Render(fmt.Sprintf("%d failed:", m.report.Failed)) + "\n")
		for _, e := range m.report.Errors {
			b.WriteString("    " + StyleDim.Render(Truncate(e, ContentWidth(m.width)-6)) + "\n")
		}
		b.WriteString("\n  " + StyleDim.Render("The full detail is in the log file.") + "\n")
	}
	return m.frame("Done", b.String(), HelpBar("enter", "back to menu", "esc", "back"))
}

// listHeight is how many rows the preview list can use.
func (m sortModel) listHeight() int {
	if m.height <= 0 {
		return 15
	}
	if h := m.height - 16; h >= 3 {
		return h
	}
	return 3
}

// window returns the slice bounds that keep cursor visible in a list of
// length total showing at most size rows.
func window(cursor, total, size int) (int, int) {
	if size >= total {
		return 0, total
	}
	start := cursor - size/2
	if start < 0 {
		start = 0
	}
	if start+size > total {
		start = total - size
	}
	return start, start + size
}

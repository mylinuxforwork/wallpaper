package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nerdexecutive/ne-image-sorter/internal/domain"
)

type menuItem struct {
	label  string
	desc   string
	screen Screen
	quit   bool
}

var menuItems = []menuItem{
	{label: "Sort Images", desc: "Scan the source, preview every decision, then move", screen: ScreenSort},
	{label: "Folders", desc: "Choose the directory to scan and the one to move into", screen: ScreenFolders},
	{label: "Rules", desc: "Edit the ordered aspect ratio and resolution policy", screen: ScreenRules},
	{label: "Quit", desc: "Exit ne-image-sorter", quit: true},
}

type menuModel struct {
	cursor int
	width  int
	cfg    domain.Config
	status string
	isErr  bool
}

func newMenuModel(cfg domain.Config) menuModel {
	return menuModel{cfg: cfg}
}

// Update handles menu input.
func (m menuModel) Update(msg tea.Msg) (menuModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch keyMsg.String() {
	case "up", "k":
		m.cursor = clampCursor(m.cursor-1, len(menuItems))
	case "down", "j":
		m.cursor = clampCursor(m.cursor+1, len(menuItems))
	case "enter":
		item := menuItems[m.cursor]
		if item.quit {
			return m, tea.Quit
		}
		m.status = ""
		return m, switchTo(item.screen)
	case "q":
		return m, tea.Quit
	}
	return m, nil
}

// View renders the main menu with the current configuration summary.
func (m menuModel) View() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(RenderBanner(m.width))
	b.WriteString("\n")
	b.WriteString(StyleSubtitle.Render("  Sort images by aspect ratio and resolution"))
	b.WriteString("\n\n")

	b.WriteString(m.summary())
	b.WriteString("\n")

	for i, item := range menuItems {
		cursor, style := "  ", StyleNormal
		if i == m.cursor {
			cursor, style = StyleSelected.Render("▸ "), StyleSelected
		}
		b.WriteString(cursor + style.Render(item.label) + "\n")
		b.WriteString("    " + StyleDim.Render(item.desc) + "\n\n")
	}

	if m.status != "" {
		style := StyleSuccess
		if m.isErr {
			style = StyleError
		}
		b.WriteString("  " + style.Render(m.status) + "\n\n")
	}

	b.WriteString(HelpBar("↑/↓", "navigate", "enter", "select", "q", "quit"))
	b.WriteString("\n")
	return b.String()
}

// summary shows the configured directories and rule count, and warns when the
// tool is not yet ready to run.
func (m menuModel) summary() string {
	w := ContentWidth(m.width)
	var b strings.Builder

	b.WriteString("  " + StyleDim.Render("source") + "  " +
		StyleNormal.Render(Truncate(orUnset(m.cfg.SourceDir), w-12)) + "\n")
	b.WriteString("  " + StyleDim.Render("move to") + " " +
		StyleNormal.Render(Truncate(orUnset(m.cfg.DestDir), w-12)) + "\n")
	b.WriteString("  " + StyleDim.Render("rules") + "   " +
		StyleNormal.Render(fmt.Sprintf("%d, then %s everything else",
			len(m.cfg.Policy.Rules), m.cfg.Policy.Default)) + "\n")

	if err := m.cfg.Validate(); err != nil {
		b.WriteString("\n  " + StyleWarning.Render("Not ready: "+err.Error()) + "\n")
		b.WriteString("  " + StyleDim.Render("Open Folders to choose a source and a destination.") + "\n")
	}
	return b.String()
}

// orUnset renders an empty path as a visible placeholder.
func orUnset(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(not set)"
	}
	return s
}

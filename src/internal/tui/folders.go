package tui

import (
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nerdexecutive/ne-image-sorter/internal/domain"
)

// foldersModel edits the source and destination paths. It carries its own
// minimal line editor rather than a widget dependency, because two single-line
// paths do not justify one.
type foldersModel struct {
	cfg    domain.Config
	field  int // 0 source, 1 destination
	values [2]string
	width  int
	err    string
}

func newFoldersModel(cfg domain.Config) foldersModel {
	return foldersModel{cfg: cfg, values: [2]string{cfg.SourceDir, cfg.DestDir}}
}

// Update handles path editing. Enter saves and returns to the menu.
func (m foldersModel) Update(msg tea.Msg) (foldersModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "esc":
		return m, switchTo(ScreenMenu)
	case "tab", "down":
		m.field = (m.field + 1) % 2
		m.err = ""
	case "shift+tab", "up":
		m.field = (m.field + 1) % 2
		m.err = ""
	case "ctrl+u":
		m.values[m.field] = ""
	case "backspace":
		if r := []rune(m.values[m.field]); len(r) > 0 {
			m.values[m.field] = string(r[:len(r)-1])
		}
	case "ctrl+d":
		// Fill the destination from the source, which is the common shape:
		// a rejects folder directly beneath the scanned directory.
		if src := strings.TrimSpace(m.values[0]); src != "" {
			m.values[1] = filepath.Join(src, "not 16-9 or 16-10 Aspect Ratio")
		}
	case "enter":
		return m.save()
	default:
		if r := []rune(keyMsg.String()); len(r) == 1 {
			m.values[m.field] += string(r)
			m.err = ""
		}
	}
	return m, nil
}

// save validates both paths and, when they hold, hands the config back.
func (m foldersModel) save() (foldersModel, tea.Cmd) {
	cfg := m.cfg
	cfg.SourceDir = strings.TrimSpace(expandHome(m.values[0]))
	cfg.DestDir = strings.TrimSpace(expandHome(m.values[1]))

	if err := cfg.Validate(); err != nil {
		m.err = err.Error()
		return m, nil
	}
	if info, err := os.Stat(cfg.SourceDir); err != nil || !info.IsDir() {
		m.err = "source directory does not exist: " + cfg.SourceDir
		return m, nil
	}

	m.cfg = cfg
	return m, tea.Batch(saveConfig(cfg), switchTo(ScreenMenu), notify("Folders saved", false))
}

// expandHome resolves a leading tilde against the user's home directory.
func expandHome(p string) string {
	p = strings.TrimSpace(p)
	if p != "~" && !strings.HasPrefix(p, "~/") {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	return filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(p, "~"), "/"))
}

// View renders the two path fields.
func (m foldersModel) View() string {
	w := ContentWidth(m.width)
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(RenderHeader("Folders", "Where to scan, and where matching images go", m.width))
	b.WriteString("\n\n")

	for i, label := range []string{"Scan this folder", "Move matches into"} {
		marker, style := "  ", StyleDim
		if i == m.field {
			marker, style = StyleSelected.Render("▸ "), StyleField
		}
		b.WriteString(marker + style.Render(label) + "\n")

		value := m.values[i]
		shown := Truncate(value, w-8)
		if i == m.field {
			shown += StyleAccent.Render("█")
		} else if strings.TrimSpace(value) == "" {
			shown = StyleDim.Render("(not set)")
		}
		b.WriteString("    " + StyleNormal.Render(shown) + "\n")
		b.WriteString("    " + StyleSeparator.Render(strings.Repeat("─", w-4)) + "\n\n")
	}

	b.WriteString("  " + StyleDim.Render("Destination is created on the first move if it is missing.") + "\n")
	if m.err != "" {
		b.WriteString("\n  " + StyleError.Render(m.err) + "\n")
	}
	b.WriteString("\n")
	b.WriteString(HelpBarWrap(m.width, "tab", "switch field", "ctrl+d", "default destination",
		"ctrl+u", "clear", "enter", "save", "esc", "cancel"))
	b.WriteString("\n")
	return b.String()
}

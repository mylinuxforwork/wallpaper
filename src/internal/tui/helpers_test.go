package tui

import (
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog"

	"github.com/nerdexecutive/ne-image-sorter/internal/domain"
	"github.com/nerdexecutive/ne-image-sorter/internal/sorter"
)

// key builds a KeyMsg for a named key or a single rune.
func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "ctrl+d":
		return tea.KeyMsg{Type: tea.KeyCtrlD}
	case "ctrl+u":
		return tea.KeyMsg{Type: tea.KeyCtrlU}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

// typeRules feeds each rune of s to a rules editor.
func typeRules(m rulesModel, s string) rulesModel {
	for _, r := range s {
		m, _ = m.Update(key(string(r)))
	}
	return m
}

// readyConfig is a configuration that passes validation.
func readyConfig() domain.Config {
	cfg := domain.DefaultConfig()
	cfg.SourceDir = "/pics"
	cfg.DestDir = "/pics/rejects"
	return cfg
}

// stubImages returns a fixed image list, so the sort screen is driven without
// a filesystem.
type stubImages struct{ images []domain.Image }

func (s stubImages) List(string, []string) ([]domain.Image, error) { return s.images, nil }
func (s stubImages) Move(domain.Image, string) (string, error)     { return "", nil }

// sortScreen builds a sort screen over a fixed image list.
func sortScreen(images ...domain.Image) sortModel {
	svc := sorter.New(stubImages{images: images}, zerolog.New(io.Discard))
	m := newSortModel(svc, readyConfig())
	m.width, m.height = 100, 40
	return m
}

// runInit executes the screen's Init command and feeds the result back, which
// is what the Bubbletea runtime does.
func runInit(m sortModel) sortModel {
	msg := m.Init()()
	m, _ = m.Update(msg)
	return m
}

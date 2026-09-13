package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Screen identifies the active screen.
type Screen int

// The screens, in menu order.
const (
	ScreenMenu Screen = iota
	ScreenFolders
	ScreenRules
	ScreenSort
)

// switchMsg asks the root model to change screen.
type switchMsg struct{ screen Screen }

// switchTo returns a command that changes screen.
func switchTo(s Screen) tea.Cmd {
	return func() tea.Msg { return switchMsg{screen: s} }
}

// statusMsg carries a one-line result back to the menu, such as a save
// confirmation, so an action always produces visible feedback.
type statusMsg struct {
	text  string
	isErr bool
}

// notify returns a command that shows text on the menu's status line.
func notify(text string, isErr bool) tea.Cmd {
	return func() tea.Msg { return statusMsg{text: text, isErr: isErr} }
}

// ContentWidth converts a terminal width into a usable content width, with a
// sane fallback before the first WindowSizeMsg arrives.
func ContentWidth(termWidth int) int {
	if termWidth <= 0 {
		return 76
	}
	if w := termWidth - 4; w >= 40 {
		return w
	}
	return 40
}

// HelpBar renders a footer from key and action pairs, for example
// HelpBar("esc", "back", "q", "quit").
func HelpBar(pairs ...string) string {
	var parts []string
	for i := 0; i+1 < len(pairs); i += 2 {
		parts = append(parts, StyleField.Render(pairs[i])+" "+pairs[i+1])
	}
	return StyleDim.Render("  " + strings.Join(parts, "  •  "))
}

// HelpBarWrap renders the same pairs as HelpBar across as many lines as the
// content width needs, so a screen with many keys stays readable at 80
// columns instead of wrapping mid-word.
func HelpBarWrap(termWidth int, pairs ...string) string {
	limit := ContentWidth(termWidth)

	var lines []string
	var current []string
	width := 2 // the leading indent

	for i := 0; i+1 < len(pairs); i += 2 {
		item := pairs[i] + " " + pairs[i+1]
		cost := len([]rune(item))
		if len(current) > 0 {
			cost += 5 // the "  •  " separator
		}
		if len(current) > 0 && width+cost > limit {
			lines = append(lines, renderHelpLine(current))
			current, width = nil, 2
			cost = len([]rune(item))
		}
		current = append(current, pairs[i], pairs[i+1])
		width += cost
	}
	if len(current) > 0 {
		lines = append(lines, renderHelpLine(current))
	}
	return strings.Join(lines, "\n")
}

// renderHelpLine formats one line of key and action pairs.
func renderHelpLine(pairs []string) string {
	var parts []string
	for i := 0; i+1 < len(pairs); i += 2 {
		parts = append(parts, StyleField.Render(pairs[i])+" "+pairs[i+1])
	}
	return StyleDim.Render("  " + strings.Join(parts, "  •  "))
}

// Truncate shortens s to max runes, ending in a single-character ellipsis.
func Truncate(s string, max int) string {
	r := []rune(s)
	if max <= 0 || len(r) <= max {
		return s
	}
	if max == 1 {
		return "…"
	}
	return string(r[:max-1]) + "…"
}

// ProgressBar renders a filled bar with a trailing percentage.
func ProgressBar(done, total, width int) string {
	if width < 10 {
		width = 10
	}
	pct := 0.0
	if total > 0 {
		pct = float64(done) / float64(total)
	}
	filled := int(pct * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	bar := StyleSuccess.Render(strings.Repeat("█", filled)) +
		StyleDim.Render(strings.Repeat("░", width-filled))
	return "[" + bar + "] " + StyleNormal.Render(pctString(pct))
}

// pctString renders a fraction as a right-aligned whole percentage.
func pctString(pct float64) string {
	n := int(pct*100 + 0.5)
	s := ""
	switch {
	case n < 10:
		s = "  "
	case n < 100:
		s = " "
	}
	return s + itoa(n) + "%"
}

// itoa converts a small non-negative int to a string without importing strconv
// for a single call site.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [4]byte
	i := len(b)
	for n > 0 && i > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// clampCursor keeps a list cursor inside [0, length).
func clampCursor(cursor, length int) int {
	if length <= 0 {
		return 0
	}
	if cursor < 0 {
		return 0
	}
	if cursor >= length {
		return length - 1
	}
	return cursor
}

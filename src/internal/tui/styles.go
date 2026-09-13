// Package tui is the Bubbletea terminal interface for ne-image-sorter.
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Palette, shared with the ne-assistant configurator so the two tools read as
// one family. Contrast ratios against a #1a1a1a terminal background:
//
//	ColorCyan     #00d7ff  ~11:1
//	ColorGreen    #00d75f  ~8.5:1
//	ColorGray     #808080  ~4.6:1  (body text, passes AA 4.5:1)
//	ColorDarkGray #6a6a6a  ~3.4:1  (separators, passes AA 3:1)
//	ColorWhite    #ffffff  ~17:1
//	ColorRed      #ff5f5f  ~5.5:1
//	ColorYellow   #ffff5f  ~16:1
//	ColorMagenta  #ff87d7  ~9:1
var (
	ColorCyan     = lipgloss.Color("#00d7ff")
	ColorGreen    = lipgloss.Color("#00d75f")
	ColorGray     = lipgloss.Color("#808080")
	ColorDarkGray = lipgloss.Color("#6a6a6a")
	ColorWhite    = lipgloss.Color("#ffffff")
	ColorRed      = lipgloss.Color("#ff5f5f")
	ColorYellow   = lipgloss.Color("#ffff5f")
	ColorMagenta  = lipgloss.Color("#ff87d7")
)

// Base styles.
var (
	StyleTitle     = lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)
	StyleSubtitle  = lipgloss.NewStyle().Foreground(ColorGray)
	StyleSelected  = lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
	StyleNormal    = lipgloss.NewStyle().Foreground(ColorWhite)
	StyleDim       = lipgloss.NewStyle().Foreground(ColorGray)
	StyleError     = lipgloss.NewStyle().Foreground(ColorRed).Bold(true)
	StyleSuccess   = lipgloss.NewStyle().Foreground(ColorGreen)
	StyleWarning   = lipgloss.NewStyle().Foreground(ColorYellow)
	StyleSeparator = lipgloss.NewStyle().Foreground(ColorDarkGray)
	StyleAccent    = lipgloss.NewStyle().Foreground(ColorMagenta)
	StyleField     = lipgloss.NewStyle().Foreground(ColorCyan)
)

// banner is the title wordmark, drawn once at the top of the main menu.
var banner = []string{
	"┌─┐┌─┐  ┬┌┬┐┌─┐┌─┐┌─┐  ┌─┐┌─┐┬─┐┌┬┐┌─┐┬─┐",
	"│││├┤   │││││├─┤│ ┬├┤   └─┐│ │├┬┘ │ ├┤ ├┬┘",
	"┘└┘└─┘  ┴┴ ┴┴ ┴└─┘└─┘  └─┘└─┘┴└─ ┴ └─┘┴└─",
}

// RenderBanner draws the wordmark above a rule, centred to the content width.
func RenderBanner(width int) string {
	w := ContentWidth(width)
	var b strings.Builder
	for _, line := range banner {
		b.WriteString(StyleTitle.Render(centre(line, w)))
		b.WriteString("\n")
	}
	b.WriteString(StyleSeparator.Render(strings.Repeat("═", w)))
	return b.String()
}

// RenderHeader draws a screen title, a rule, and an optional subtitle.
func RenderHeader(title, subtitle string, width int) string {
	w := ContentWidth(width)
	var b strings.Builder
	b.WriteString(StyleTitle.Render(title))
	b.WriteString("\n")
	b.WriteString(StyleSeparator.Render(strings.Repeat("═", w)))
	if subtitle != "" {
		b.WriteString("\n")
		b.WriteString(StyleSubtitle.Render(subtitle))
	}
	return b.String()
}

// centre pads s with leading spaces so it sits in the middle of width. Rune
// count is used rather than byte length because the wordmark is box-drawing.
func centre(s string, width int) string {
	pad := (width - len([]rune(s))) / 2
	if pad <= 0 {
		return s
	}
	return strings.Repeat(" ", pad) + s
}

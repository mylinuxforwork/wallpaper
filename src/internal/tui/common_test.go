package tui

import (
	"strings"
	"testing"
)

func TestContentWidth(t *testing.T) {
	tests := []struct {
		in, want int
	}{
		{0, 76}, {-5, 76}, {84, 80}, {44, 40}, {20, 40}, {200, 196},
	}
	for _, tc := range tests {
		if got := ContentWidth(tc.in); got != tc.want {
			t.Errorf("ContentWidth(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		in   string
		max  int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello", 4, "hel…"},
		{"hello", 1, "…"},
		{"hello", 0, "hello"},
		{"héllo wörld", 6, "héllo…"},
	}
	for _, tc := range tests {
		if got := Truncate(tc.in, tc.max); got != tc.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tc.in, tc.max, got, tc.want)
		}
	}
}

func TestProgressBarBounds(t *testing.T) {
	// Must not panic or overflow at the edges, including a zero total.
	for _, tc := range []struct{ done, total int }{{0, 0}, {0, 10}, {5, 10}, {10, 10}, {20, 10}, {-1, 10}} {
		out := ProgressBar(tc.done, tc.total, 20)
		if !strings.Contains(out, "%") {
			t.Errorf("ProgressBar(%d, %d) = %q, want a percentage", tc.done, tc.total, out)
		}
	}
}

func TestProgressBarPercentages(t *testing.T) {
	tests := []struct {
		done, total int
		want        string
	}{
		{0, 10, "  0%"}, {5, 10, " 50%"}, {10, 10, "100%"}, {0, 0, "  0%"},
	}
	for _, tc := range tests {
		if got := ProgressBar(tc.done, tc.total, 20); !strings.Contains(got, tc.want) {
			t.Errorf("ProgressBar(%d, %d) = %q, want it to contain %q", tc.done, tc.total, got, tc.want)
		}
	}
}

func TestClampCursor(t *testing.T) {
	tests := []struct {
		cursor, length, want int
	}{
		{0, 5, 0}, {4, 5, 4}, {5, 5, 4}, {99, 5, 4}, {-1, 5, 0}, {3, 0, 0}, {0, 0, 0},
	}
	for _, tc := range tests {
		if got := clampCursor(tc.cursor, tc.length); got != tc.want {
			t.Errorf("clampCursor(%d, %d) = %d, want %d", tc.cursor, tc.length, got, tc.want)
		}
	}
}

func TestHelpBarPairs(t *testing.T) {
	out := HelpBar("esc", "back", "q", "quit")
	for _, want := range []string{"esc", "back", "q", "quit", "•"} {
		if !strings.Contains(out, want) {
			t.Errorf("HelpBar() = %q, want it to contain %q", out, want)
		}
	}
	// An odd trailing key with no action is dropped rather than rendered bare.
	if strings.Contains(HelpBar("esc", "back", "x"), "x") {
		t.Error("HelpBar() rendered an unpaired key")
	}
}

func TestHelpBarWrapFitsTheWidth(t *testing.T) {
	pairs := []string{"↑/↓", "select", "enter", "edit", "a", "add", "d", "delete",
		"J/K", "reorder", "t", "toggle default", "r", "reset", "esc", "back"}

	out := HelpBarWrap(84, pairs...)
	for _, line := range strings.Split(out, "\n") {
		if n := len([]rune(stripANSI(line))); n > 80 {
			t.Errorf("help line is %d runes wide, over the 80 content width: %q", n, line)
		}
	}
	// Every key and action must survive the wrap.
	for _, want := range pairs {
		if !strings.Contains(out, want) {
			t.Errorf("HelpBarWrap dropped %q", want)
		}
	}
}

func TestHelpBarWrapSingleLineWhenItFits(t *testing.T) {
	out := HelpBarWrap(84, "esc", "back", "q", "quit")
	if strings.Contains(out, "\n") {
		t.Errorf("a short help bar must stay on one line, got %q", out)
	}
}

// stripANSI removes escape sequences so a rendered line can be measured.
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			i++
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func TestRenderHeaderAndBanner(t *testing.T) {
	if out := RenderHeader("Title", "Sub", 84); !strings.Contains(out, "Title") || !strings.Contains(out, "Sub") {
		t.Errorf("RenderHeader() = %q, want the title and subtitle", out)
	}
	if out := RenderHeader("Title", "", 84); !strings.Contains(out, strings.Repeat("═", 80)) {
		t.Error("RenderHeader() separator width must follow ContentWidth")
	}
	// Neither may panic before the first window size arrives.
	if out := RenderBanner(0); out == "" {
		t.Error("RenderBanner(0) returned nothing")
	}
}

func TestCentreNeverTruncates(t *testing.T) {
	long := strings.Repeat("x", 100)
	if got := centre(long, 20); got != long {
		t.Error("centre() must not alter a string wider than the target width")
	}
}

// TestPaletteContrastRegression pins the two colours whose contrast is closest
// to the WCAG AA floor, so a later palette tweak that fails accessibility
// breaks the build rather than shipping.
func TestPaletteContrastRegression(t *testing.T) {
	if string(ColorGray) != "#808080" {
		t.Errorf("ColorGray = %s; body text must stay at #808080 (4.6:1) or lighter", ColorGray)
	}
	if string(ColorDarkGray) == "#444444" {
		t.Error("ColorDarkGray is #444444, which fails WCAG AA 3:1 for UI components")
	}
}

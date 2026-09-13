package tui

import (
	"io"
	"strings"
	"testing"

	"github.com/nerdexecutive/ne-image-sorter/internal/domain"
	"github.com/nerdexecutive/ne-image-sorter/internal/sorter"
)

func TestSortScanThenPreview(t *testing.T) {
	m := runInit(sortScreen(
		domain.Image{Name: "uhd.jpg", Width: 3840, Height: 2160},
		domain.Image{Name: "wide.jpg", Width: 3440, Height: 1440},
	))
	if m.stage != stagePreview {
		t.Fatalf("stage = %v after the scan, want preview", m.stage)
	}
	if m.plan.MoveCount != 1 || m.plan.KeepCount != 1 {
		t.Errorf("plan = %d move, %d keep; want 1 and 1", m.plan.MoveCount, m.plan.KeepCount)
	}

	out := m.View()
	for _, want := range []string{"1 to move", "1 to keep", "wide.jpg"} {
		if !strings.Contains(out, want) {
			t.Errorf("preview is missing %q", want)
		}
	}
	// The keep decision is hidden until the filter is toggled.
	if strings.Contains(out, "uhd.jpg") {
		t.Error("preview must show moves only until v is pressed")
	}
}

func TestSortPreviewShowsTheDecidingReason(t *testing.T) {
	m := runInit(sortScreen(domain.Image{Name: "hd.jpg", Width: 1920, Height: 1080}))
	if out := m.View(); !strings.Contains(out, "height") {
		t.Error("the preview must name the rule that decided the selected image")
	}
}

func TestSortFilterTogglesKeeps(t *testing.T) {
	m := runInit(sortScreen(
		domain.Image{Name: "uhd.jpg", Width: 3840, Height: 2160},
		domain.Image{Name: "wide.jpg", Width: 3440, Height: 1440},
	))
	m, _ = m.Update(key("v"))
	if !strings.Contains(m.View(), "uhd.jpg") {
		t.Error("v must reveal the images that are staying")
	}
}

func TestSortRequiresConfirmationBeforeMoving(t *testing.T) {
	m := runInit(sortScreen(domain.Image{Name: "wide.jpg", Width: 3440, Height: 1440}))

	m, cmd := m.Update(key("enter"))
	if m.stage != stageConfirm {
		t.Fatalf("stage = %v after enter, want confirm", m.stage)
	}
	if cmd != nil {
		t.Error("enter must not start moving files; it must ask first")
	}
	if !strings.Contains(m.View(), "Move 1 image(s)?") {
		t.Error("the confirm screen must state how many files move")
	}

	// n returns to the preview without moving anything.
	back, cmd := m.Update(key("n"))
	if back.stage != stagePreview || cmd != nil {
		t.Error("n must return to the preview without moving")
	}

	// y is the only key that starts the move.
	started, cmd := m.Update(key("y"))
	if cmd == nil || started.stage != stageScanning {
		t.Error("y must start the move")
	}
}

func TestSortConfirmIgnoresOtherKeys(t *testing.T) {
	m := runInit(sortScreen(domain.Image{Name: "wide.jpg", Width: 3440, Height: 1440}))
	m, _ = m.Update(key("enter"))
	for _, k := range []string{"enter", " ", "j", "d"} {
		next, cmd := m.Update(key(k))
		if cmd != nil || next.stage != stageConfirm {
			t.Errorf("key %q on the confirm screen must not move files", k)
		}
	}
}

func TestSortWithNothingToMoveCannotBeConfirmed(t *testing.T) {
	m := runInit(sortScreen(domain.Image{Name: "uhd.jpg", Width: 3840, Height: 2160}))
	m, _ = m.Update(key("enter"))
	if m.stage == stageConfirm {
		t.Error("an empty move set must not open the confirm screen")
	}
	if !strings.Contains(m.View(), "Nothing to move") {
		t.Error("an empty move set must render an explicit empty state")
	}
}

func TestSortEmptySourceRenders(t *testing.T) {
	m := runInit(sortScreen())
	if out := m.View(); !strings.Contains(out, "0 scanned") {
		t.Errorf("an empty source must report zero scanned, got %q", out)
	}
}

func TestSortReportsFailures(t *testing.T) {
	m := sortScreen()
	m.plan = sorter.Plan{MoveCount: 3}
	m, _ = m.Update(reportMsg{report: sorter.Report{Moved: 2, Failed: 1, Errors: []string{"b.jpg: denied"}}})

	if m.stage != stageDone {
		t.Fatalf("stage = %v, want done", m.stage)
	}
	out := m.View()
	for _, want := range []string{"Moved 2", "1 failed", "b.jpg"} {
		if !strings.Contains(out, want) {
			t.Errorf("the done screen is missing %q", want)
		}
	}
}

func TestSortSurfacesAScanError(t *testing.T) {
	m := sortScreen()
	m, _ = m.Update(planMsg{err: io.ErrUnexpectedEOF})
	if m.stage != stageFailed {
		t.Fatalf("stage = %v, want failed", m.stage)
	}
	if out := m.View(); !strings.Contains(out, "Sort failed") || !strings.Contains(out, "Folders") {
		t.Error("a failure must name the problem and suggest the fix")
	}
}

// TestEscapeAlwaysLeavesTheSortScreen pins the keymap consistency rule: esc
// means back on every stage, never something else.
func TestEscapeAlwaysLeavesTheSortScreen(t *testing.T) {
	for name, stage := range map[string]sortStage{
		"scanning": stageScanning, "preview": stagePreview,
		"confirm": stageConfirm, "done": stageDone, "failed": stageFailed,
	} {
		m := sortScreen()
		m.stage = stage
		if _, cmd := m.Update(key("esc")); cmd == nil {
			t.Errorf("esc on the %s stage produced no command, want back", name)
		}
	}
}

func TestSortNavigationStaysInBounds(t *testing.T) {
	m := runInit(sortScreen(
		domain.Image{Name: "a.jpg", Width: 3440, Height: 1440},
		domain.Image{Name: "b.jpg", Width: 3440, Height: 1440},
	))
	for i := 0; i < 10; i++ {
		m, _ = m.Update(key("down"))
	}
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want it clamped to 1", m.cursor)
	}
	for i := 0; i < 10; i++ {
		m, _ = m.Update(key("up"))
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want it clamped to 0", m.cursor)
	}
}

func TestWindowKeepsTheCursorVisible(t *testing.T) {
	tests := []struct {
		cursor, total, size int
	}{
		{0, 100, 10}, {50, 100, 10}, {99, 100, 10}, {5, 3, 10}, {0, 0, 10},
	}
	for _, tc := range tests {
		start, end := window(tc.cursor, tc.total, tc.size)
		if start < 0 || end > tc.total || start > end {
			t.Fatalf("window(%d,%d,%d) = %d,%d is out of range", tc.cursor, tc.total, tc.size, start, end)
		}
		if tc.total > 0 && tc.cursor < tc.total && (tc.cursor < start || tc.cursor >= end) {
			t.Errorf("window(%d,%d,%d) = %d,%d excludes the cursor", tc.cursor, tc.total, tc.size, start, end)
		}
	}
}

func TestListHeightHasAFloor(t *testing.T) {
	for _, h := range []int{0, -1, 5, 40} {
		m := sortScreen()
		m.height = h
		if got := m.listHeight(); got < 3 {
			t.Errorf("listHeight() at terminal height %d = %d, want at least 3", h, got)
		}
	}
}

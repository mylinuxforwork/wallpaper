package tui

import (
	"strings"
	"testing"

	"github.com/nerdexecutive/ne-image-sorter/internal/domain"
)

func TestRulesReorderChangesPrecedence(t *testing.T) {
	m := newRulesModel(readyConfig())
	first := m.cfg.Policy.Rules[0]

	m, _ = m.Update(key("J")) // move rule 1 down
	if m.cfg.Policy.Rules[1] != first {
		t.Error("J must move the selected rule down one place")
	}
	if m.cursor != 1 {
		t.Errorf("cursor = %d after J, want it to follow the rule to 1", m.cursor)
	}

	m, _ = m.Update(key("K"))
	if m.cfg.Policy.Rules[0] != first {
		t.Error("K must move the selected rule back up")
	}
}

func TestRulesReorderAtTheEdgesIsANoOp(t *testing.T) {
	m := newRulesModel(readyConfig())
	before := len(m.cfg.Policy.Rules)
	m, _ = m.Update(key("K")) // already at the top
	if len(m.cfg.Policy.Rules) != before || m.cursor != 0 {
		t.Error("K at the top must change nothing")
	}
}

func TestRulesDeleteAndReset(t *testing.T) {
	m := newRulesModel(readyConfig())
	original := len(m.cfg.Policy.Rules)

	m, cmd := m.Update(key("d"))
	if cmd == nil {
		t.Error("delete must persist the change")
	}
	if len(m.cfg.Policy.Rules) != original-1 {
		t.Fatalf("after delete there are %d rules, want %d", len(m.cfg.Policy.Rules), original-1)
	}

	m, _ = m.Update(key("r"))
	if len(m.cfg.Policy.Rules) != original {
		t.Errorf("reset gave %d rules, want the shipped %d", len(m.cfg.Policy.Rules), original)
	}
}

func TestRulesToggleDefault(t *testing.T) {
	m := newRulesModel(readyConfig())
	if m.cfg.Policy.Default != domain.ActionMove {
		t.Fatal("the shipped default must be move")
	}
	m, _ = m.Update(key("t"))
	if m.cfg.Policy.Default != domain.ActionKeep {
		t.Errorf("default = %v after toggle, want keep", m.cfg.Policy.Default)
	}
	m, _ = m.Update(key("t"))
	if m.cfg.Policy.Default != domain.ActionMove {
		t.Error("toggling twice must return to move")
	}
}

func TestRulesAddARatioRule(t *testing.T) {
	m := newRulesModel(readyConfig())
	original := len(m.cfg.Policy.Rules)

	m, _ = m.Update(key("a"))
	if m.mode != modeEdit || m.editIdx != -1 {
		t.Fatal("a must open the editor in add mode")
	}

	// Property defaults to aspect. Move to the value row and type a ratio.
	m, _ = m.Update(key("down"))
	m, _ = m.Update(key("down"))
	m = typeRules(m, "21:9")
	m, cmd := m.Update(key("enter"))

	if cmd == nil {
		t.Fatalf("enter did not commit, err = %q", m.err)
	}
	if len(m.cfg.Policy.Rules) != original+1 {
		t.Fatalf("rule count = %d, want %d", len(m.cfg.Policy.Rules), original+1)
	}
	added := m.cfg.Policy.Rules[original]
	if added.Label != "21:9" {
		t.Errorf("label = %q, want the typed ratio preserved", added.Label)
	}
	if diff := added.Value - 21.0/9.0; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("value = %v, want 21/9", added.Value)
	}
	if added.Tolerance != 0.01 {
		t.Errorf("tolerance = %v, want the 1%% default", added.Tolerance)
	}
}

func TestRulesRejectsABadRatio(t *testing.T) {
	m := newRulesModel(readyConfig())
	m, _ = m.Update(key("a"))
	m, _ = m.Update(key("down"))
	m, _ = m.Update(key("down"))
	m = typeRules(m, "wide")
	m, cmd := m.Update(key("enter"))

	if cmd != nil {
		t.Error("a bad ratio must not commit")
	}
	if !strings.Contains(m.err, "16:9") {
		t.Errorf("err = %q, want it to suggest a valid form", m.err)
	}
}

func TestRulesRejectsABadTolerance(t *testing.T) {
	m := newRulesModel(readyConfig())
	m, _ = m.Update(key("a"))
	m, _ = m.Update(key("down"))
	m, _ = m.Update(key("down"))
	m = typeRules(m, "16:9")
	m.tolBuf = "abc"
	m, cmd := m.Update(key("enter"))

	if cmd != nil || !strings.Contains(m.err, "percentage") {
		t.Errorf("a bad tolerance must be refused, got cmd=%v err=%q", cmd, m.err)
	}
}

func TestRulesEmptyToleranceMeansExact(t *testing.T) {
	m := newRulesModel(readyConfig())
	m, _ = m.Update(key("a"))
	m, _ = m.Update(key("down"))
	m, _ = m.Update(key("down"))
	m = typeRules(m, "16:9")
	m.tolBuf = ""
	m, cmd := m.Update(key("enter"))

	if cmd == nil {
		t.Fatalf("an empty tolerance must commit as exact, err = %q", m.err)
	}
	if added := m.cfg.Policy.Rules[len(m.cfg.Policy.Rules)-1]; added.Tolerance != 0 {
		t.Errorf("tolerance = %v, want 0", added.Tolerance)
	}
}

func TestRulesCyclingWrapsBothWays(t *testing.T) {
	m := newRulesModel(readyConfig())
	m, _ = m.Update(key("a"))

	fields := domain.Fields()
	for i := 0; i < len(fields); i++ {
		m, _ = m.Update(key("right"))
	}
	if m.editing.Field != fields[0] {
		t.Errorf("cycling a full turn gave %v, want %v", m.editing.Field, fields[0])
	}

	m, _ = m.Update(key("left"))
	if m.editing.Field != fields[len(fields)-1] {
		t.Errorf("cycling left from the first gave %v, want the last", m.editing.Field)
	}
}

func TestRulesPixelRuleTakesAWholeNumber(t *testing.T) {
	m := newRulesModel(readyConfig())
	m, _ = m.Update(key("a"))
	m, _ = m.Update(key("right")) // aspect -> width
	m, _ = m.Update(key("right")) // width  -> height
	m, _ = m.Update(key("down"))  // operator row
	m, _ = m.Update(key("right")) // == -> !=
	m, _ = m.Update(key("down"))  // value row
	m = typeRules(m, "1440")
	m, cmd := m.Update(key("enter"))

	if cmd == nil {
		t.Fatalf("a pixel rule did not commit, err = %q", m.err)
	}
	added := m.cfg.Policy.Rules[len(m.cfg.Policy.Rules)-1]
	if added.Field != domain.FieldHeight || added.Value != 1440 {
		t.Errorf("added %+v, want height 1440", added)
	}
	if added.Label != "" {
		t.Errorf("label = %q, want none on a pixel rule", added.Label)
	}
}

func TestRulesEscapeLeavesTheEditorWithoutSaving(t *testing.T) {
	m := newRulesModel(readyConfig())
	original := len(m.cfg.Policy.Rules)
	m, _ = m.Update(key("a"))
	m, _ = m.Update(key("esc"))
	if m.mode != modeList {
		t.Error("esc must return to the list")
	}
	if len(m.cfg.Policy.Rules) != original {
		t.Error("esc must not add the half-built rule")
	}
}

func TestRulesEditPreservesTheRatioLabel(t *testing.T) {
	m := newRulesModel(readyConfig())
	m.cursor = 1 // the 16:9 keep rule
	m, _ = m.Update(key("enter"))
	if m.valueBuf != "16:9" {
		t.Errorf("editor opened with value %q, want the stored label 16:9", m.valueBuf)
	}
	if m.tolBuf != "1" {
		t.Errorf("editor opened with tolerance %q, want 1", m.tolBuf)
	}
}

func TestRulesViewRendersEveryRuleAndTheDefault(t *testing.T) {
	out := newRulesModel(readyConfig()).viewList()
	for _, want := range []string{"1080", "16:9", "16:10", "otherwise", "move"} {
		if !strings.Contains(out, want) {
			t.Errorf("rules list is missing %q", want)
		}
	}
}

func TestRulesEmptyListRenders(t *testing.T) {
	cfg := readyConfig()
	cfg.Policy.Rules = nil
	out := newRulesModel(cfg).viewList()
	if !strings.Contains(out, "No rules") {
		t.Error("an empty policy must render an empty state, not a blank screen")
	}
}

func TestRulesEditorViewRenders(t *testing.T) {
	m := newRulesModel(readyConfig())
	m, _ = m.Update(key("a"))
	out := m.View()
	for _, want := range []string{"Add Rule", "Property", "Comparison", "Tolerance", "aspect"} {
		if !strings.Contains(out, want) {
			t.Errorf("rule editor is missing %q", want)
		}
	}
}

func TestWrap(t *testing.T) {
	tests := []struct{ i, n, want int }{
		{0, 3, 0}, {2, 3, 2}, {3, 3, 0}, {-1, 3, 2}, {-4, 3, 2}, {1, 0, 0},
	}
	for _, tc := range tests {
		if got := wrap(tc.i, tc.n); got != tc.want {
			t.Errorf("wrap(%d, %d) = %d, want %d", tc.i, tc.n, got, tc.want)
		}
	}
}

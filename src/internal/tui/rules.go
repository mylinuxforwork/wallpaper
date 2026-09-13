package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nerdexecutive/ne-image-sorter/internal/domain"
)

// rulesMode is the rules screen's sub-state: browsing the ordered list, or
// editing one rule's fields.
type rulesMode int

const (
	modeList rulesMode = iota
	modeEdit
)

// editField indexes the rule editor's rows.
type editField int

const (
	fieldProperty editField = iota
	fieldOperator
	fieldValue
	fieldTolerance
	fieldAction
	fieldCount
)

var editLabels = [fieldCount]string{"Property", "Comparison", "Value", "Tolerance", "Then"}

// rulesModel edits the ordered policy.
type rulesModel struct {
	cfg      domain.Config
	cursor   int
	mode     rulesMode
	editing  domain.Rule
	editIdx  int // -1 when adding
	editRow  editField
	valueBuf string
	tolBuf   string
	width    int
	err      string
}

func newRulesModel(cfg domain.Config) rulesModel {
	return rulesModel{cfg: cfg, editIdx: -1}
}

// Update routes input to the list or the editor.
func (m rulesModel) Update(msg tea.Msg) (rulesModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	if m.mode == modeEdit {
		return m.updateEdit(keyMsg)
	}
	return m.updateList(keyMsg)
}

// updateList handles browsing, reordering, deleting, and the default action.
func (m rulesModel) updateList(k tea.KeyMsg) (rulesModel, tea.Cmd) {
	rules := m.cfg.Policy.Rules
	switch k.String() {
	case "esc":
		return m, switchTo(ScreenMenu)
	case "up", "k":
		m.cursor = clampCursor(m.cursor-1, len(rules))
	case "down", "j":
		m.cursor = clampCursor(m.cursor+1, len(rules))
	case "K":
		m = m.swap(m.cursor, m.cursor-1)
	case "J":
		m = m.swap(m.cursor, m.cursor+1)
	case "a":
		m.mode, m.editIdx, m.editRow = modeEdit, -1, fieldProperty
		m.editing = domain.Rule{Field: domain.FieldAspect, Op: domain.OpEqual, Action: domain.ActionKeep}
		m.valueBuf, m.tolBuf, m.err = "", "1", ""
	case "enter":
		if len(rules) == 0 {
			return m, nil
		}
		m.mode, m.editIdx, m.editRow = modeEdit, m.cursor, fieldProperty
		m.editing = rules[m.cursor]
		m.valueBuf = m.editing.Label
		if m.valueBuf == "" {
			m.valueBuf = trimFloat(m.editing.Value)
		}
		m.tolBuf, m.err = trimFloat(m.editing.Tolerance*100), ""
	case "d":
		if len(rules) > 0 {
			m.cfg.Policy.Rules = append(append([]domain.Rule{}, rules[:m.cursor]...), rules[m.cursor+1:]...)
			m.cursor = clampCursor(m.cursor, len(m.cfg.Policy.Rules))
			return m, saveConfig(m.cfg)
		}
	case "t":
		// Toggle the fallback that applies when no rule matches.
		if m.cfg.Policy.Default == domain.ActionMove {
			m.cfg.Policy.Default = domain.ActionKeep
		} else {
			m.cfg.Policy.Default = domain.ActionMove
		}
		return m, saveConfig(m.cfg)
	case "r":
		m.cfg.Policy = domain.DefaultPolicy()
		m.cursor = 0
		return m, saveConfig(m.cfg)
	}
	return m, nil
}

// swap moves the rule at i to position j, which changes which rule decides
// first and so can change the outcome.
func (m rulesModel) swap(i, j int) rulesModel {
	rules := m.cfg.Policy.Rules
	if i < 0 || j < 0 || i >= len(rules) || j >= len(rules) {
		return m
	}
	rules[i], rules[j] = rules[j], rules[i]
	m.cursor = j
	return m
}

// updateEdit handles the rule editor.
func (m rulesModel) updateEdit(k tea.KeyMsg) (rulesModel, tea.Cmd) {
	switch k.String() {
	case "esc":
		m.mode, m.err = modeList, ""
		return m, nil
	case "up":
		m.editRow = editField(clampCursor(int(m.editRow)-1, int(fieldCount)))
	case "down", "tab":
		m.editRow = editField(clampCursor(int(m.editRow)+1, int(fieldCount)))
	case "left":
		m = m.cycle(-1)
	case "right":
		m = m.cycle(1)
	case "backspace":
		m = m.editText(func(s string) string {
			if r := []rune(s); len(r) > 0 {
				return string(r[:len(r)-1])
			}
			return s
		})
	case "enter":
		return m.commit()
	default:
		if r := []rune(k.String()); len(r) == 1 {
			m = m.editText(func(s string) string { return s + string(r) })
		}
	}
	return m, nil
}

// cycle steps an enumerated field left or right.
func (m rulesModel) cycle(delta int) rulesModel {
	switch m.editRow {
	case fieldProperty:
		fields := domain.Fields()
		m.editing.Field = fields[wrap(indexOfField(fields, m.editing.Field)+delta, len(fields))]
	case fieldOperator:
		ops := domain.Operators()
		m.editing.Op = ops[wrap(indexOfOp(ops, m.editing.Op)+delta, len(ops))]
	case fieldAction:
		if m.editing.Action == domain.ActionMove {
			m.editing.Action = domain.ActionKeep
		} else {
			m.editing.Action = domain.ActionMove
		}
	}
	m.err = ""
	return m
}

// editText applies edit to whichever text buffer the cursor is on.
func (m rulesModel) editText(edit func(string) string) rulesModel {
	switch m.editRow {
	case fieldValue:
		m.valueBuf = edit(m.valueBuf)
	case fieldTolerance:
		m.tolBuf = edit(m.tolBuf)
	default:
		return m
	}
	m.err = ""
	return m
}

// commit parses the buffers into the rule and stores it.
func (m rulesModel) commit() (rulesModel, tea.Cmd) {
	rule := m.editing

	if rule.Field == domain.FieldAspect {
		v, err := domain.ParseRatio(m.valueBuf)
		if err != nil {
			m.err = "value must be a ratio such as 16:9, or a decimal such as 1.7778"
			return m, nil
		}
		rule.Value = v
		rule.Label = strings.TrimSpace(m.valueBuf)
		if !strings.ContainsAny(rule.Label, ":/") {
			rule.Label = ""
		}
	} else {
		v, err := strconv.ParseFloat(strings.TrimSpace(m.valueBuf), 64)
		if err != nil || v < 0 {
			m.err = "value must be a pixel count, such as 1080"
			return m, nil
		}
		rule.Value, rule.Label = v, ""
	}

	pct, err := strconv.ParseFloat(strings.TrimSpace(orZero(m.tolBuf)), 64)
	if err != nil || pct < 0 {
		m.err = "tolerance must be a percentage, such as 1 or 2.5"
		return m, nil
	}
	rule.Tolerance = pct / 100

	if m.editIdx < 0 {
		m.cfg.Policy.Rules = append(m.cfg.Policy.Rules, rule)
		m.cursor = len(m.cfg.Policy.Rules) - 1
	} else {
		m.cfg.Policy.Rules[m.editIdx] = rule
	}
	m.mode, m.err = modeList, ""
	return m, saveConfig(m.cfg)
}

// View renders the list or the editor.
func (m rulesModel) View() string {
	if m.mode == modeEdit {
		return m.viewEdit()
	}
	return m.viewList()
}

// viewList renders the ordered policy.
func (m rulesModel) viewList() string {
	w := ContentWidth(m.width)
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(RenderHeader("Rules", "Checked top to bottom. The first match decides.", m.width))
	b.WriteString("\n\n")

	if len(m.cfg.Policy.Rules) == 0 {
		b.WriteString("  " + StyleWarning.Render("No rules. Every image falls through to the default below.") + "\n")
		b.WriteString("  " + StyleDim.Render("Press a to add one, or r to restore the shipped policy.") + "\n\n")
	}

	for i, rule := range m.cfg.Policy.Rules {
		marker, style := "  ", StyleNormal
		if i == m.cursor {
			marker, style = StyleSelected.Render("▸ "), StyleSelected
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", marker,
			StyleDim.Render(fmt.Sprintf("%d.", i+1)),
			style.Render(Truncate(rule.Describe(), w-8))))
	}

	b.WriteString("\n  " + StyleSeparator.Render(strings.Repeat("─", w-2)) + "\n")
	b.WriteString("  " + StyleDim.Render("otherwise") + "  " +
		actionStyle(m.cfg.Policy.Default).Render(string(m.cfg.Policy.Default)) + "\n\n")

	b.WriteString(HelpBarWrap(m.width, "↑/↓", "select", "enter", "edit", "a", "add",
		"d", "delete", "J/K", "reorder", "t", "toggle default", "r", "reset", "esc", "back"))
	b.WriteString("\n")
	return b.String()
}

// viewEdit renders the single-rule editor.
func (m rulesModel) viewEdit() string {
	w := ContentWidth(m.width)
	title := "Edit Rule"
	if m.editIdx < 0 {
		title = "Add Rule"
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(RenderHeader(title, "← → change a choice, type to edit a value", m.width))
	b.WriteString("\n\n")

	values := [fieldCount]string{
		string(m.editing.Field),
		string(m.editing.Op) + "  " + StyleDim.Render(opHint(m.editing.Op)),
		valueOrHint(m.valueBuf, m.editing.Field) + StyleAccent.Render(cursorIf(m.editRow == fieldValue)),
		m.tolBuf + "%" + StyleAccent.Render(cursorIf(m.editRow == fieldTolerance)),
		string(m.editing.Action) + "  " + StyleDim.Render(actionHint(m.editing.Action)),
	}

	for i := range values {
		marker, label := "  ", StyleDim
		if editField(i) == m.editRow {
			marker, label = StyleSelected.Render("▸ "), StyleField
		}
		b.WriteString(fmt.Sprintf("%s%s  %s\n", marker,
			label.Render(fmt.Sprintf("%-11s", editLabels[i])), values[i]))
	}

	b.WriteString("\n  " + StyleSeparator.Render(strings.Repeat("─", w-2)) + "\n")
	if m.editing.Field == domain.FieldAspect {
		b.WriteString("  " + StyleDim.Render("Ratios accept 16:9, 16/10, or a decimal like 1.7778.") + "\n")
	} else {
		b.WriteString("  " + StyleDim.Render("Values are pixels. Tolerance applies to == and != only.") + "\n")
	}
	if m.err != "" {
		b.WriteString("\n  " + StyleError.Render(m.err) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(HelpBar("↑/↓", "field", "←/→", "change", "enter", "save", "esc", "cancel"))
	b.WriteString("\n")
	return b.String()
}

// opHint explains an operator in words.
func opHint(op domain.Operator) string {
	switch op {
	case domain.OpEqual:
		return "matches, within tolerance"
	case domain.OpNotEqual:
		return "differs, beyond tolerance"
	case domain.OpLess:
		return "is under"
	case domain.OpLessEqual:
		return "is at most"
	case domain.OpGreater:
		return "is over"
	case domain.OpGreaterEqual:
		return "is at least"
	default:
		return ""
	}
}

// actionHint explains an action in words.
func actionHint(a domain.Action) string {
	if a == domain.ActionMove {
		return "relocate to the destination folder"
	}
	return "leave in the source folder"
}

// actionStyle colours a verdict: green keeps, yellow moves.
func actionStyle(a domain.Action) interface{ Render(...string) string } {
	if a == domain.ActionMove {
		return StyleWarning
	}
	return StyleSuccess
}

// valueOrHint renders the typed value, or a dim example when it is empty, so
// the field never reads as broken.
func valueOrHint(buf string, field domain.Field) string {
	if strings.TrimSpace(buf) != "" {
		return buf
	}
	if field == domain.FieldAspect {
		return StyleDim.Render("16:9")
	}
	return StyleDim.Render("1080")
}

// cursorIf returns a block cursor when the field has focus.
func cursorIf(focused bool) string {
	if focused {
		return "█"
	}
	return ""
}

// trimFloat renders a float without trailing zeros.
func trimFloat(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

// orZero substitutes "0" for an empty buffer.
func orZero(s string) string {
	if strings.TrimSpace(s) == "" {
		return "0"
	}
	return s
}

// wrap returns i modulo n, wrapping negatives around.
func wrap(i, n int) int {
	if n <= 0 {
		return 0
	}
	return ((i % n) + n) % n
}

func indexOfField(fields []domain.Field, f domain.Field) int {
	for i, v := range fields {
		if v == f {
			return i
		}
	}
	return 0
}

func indexOfOp(ops []domain.Operator, o domain.Operator) int {
	for i, v := range ops {
		if v == o {
			return i
		}
	}
	return 0
}

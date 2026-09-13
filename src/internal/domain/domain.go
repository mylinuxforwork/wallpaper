// Package domain holds the sorting vocabulary: the image record, the rule
// language that classifies one, and the configuration that carries a policy.
// It depends on nothing outside the standard library so the policy stays
// testable without a filesystem.
package domain

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Image is one discovered image file and its pixel dimensions.
type Image struct {
	Path   string `json:"path"`
	Name   string `json:"name"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// Aspect returns the width-to-height ratio, or 0 when the height is unusable.
func (i Image) Aspect() float64 {
	if i.Height <= 0 {
		return 0
	}
	return float64(i.Width) / float64(i.Height)
}

// Resolution renders the dimensions for display, for example "3840x2160".
func (i Image) Resolution() string {
	return fmt.Sprintf("%dx%d", i.Width, i.Height)
}

// Field names the image property a rule tests.
type Field string

// The testable image properties.
const (
	FieldAspect Field = "aspect"
	FieldWidth  Field = "width"
	FieldHeight Field = "height"
)

// Fields lists every valid field in menu order.
func Fields() []Field { return []Field{FieldAspect, FieldWidth, FieldHeight} }

// Operator is the comparison a rule applies to a field.
type Operator string

// The supported comparisons. Tolerance applies to OpEqual and OpNotEqual only.
const (
	OpEqual        Operator = "=="
	OpNotEqual     Operator = "!="
	OpLess         Operator = "<"
	OpLessEqual    Operator = "<="
	OpGreater      Operator = ">"
	OpGreaterEqual Operator = ">="
)

// Operators lists every valid operator in menu order.
func Operators() []Operator {
	return []Operator{OpEqual, OpNotEqual, OpLess, OpLessEqual, OpGreater, OpGreaterEqual}
}

// Action is the verdict a rule or a policy default produces.
type Action string

// The two verdicts. Move relocates the file; Keep leaves it in the source.
const (
	ActionMove Action = "move"
	ActionKeep Action = "keep"
)

// Rule is one ordered policy entry: a comparison against a field, and the
// verdict to return when it matches. Tolerance is a fraction of Value, so
// 0.01 means one percent, and it is read only by OpEqual and OpNotEqual.
type Rule struct {
	Field     Field    `json:"field"`
	Op        Operator `json:"op"`
	Value     float64  `json:"value"`
	Tolerance float64  `json:"tolerance"`
	Action    Action   `json:"action"`
	Label     string   `json:"label,omitempty"`
}

// actual reads the field this rule tests from img.
func (r Rule) actual(img Image) (float64, bool) {
	switch r.Field {
	case FieldAspect:
		return img.Aspect(), true
	case FieldWidth:
		return float64(img.Width), true
	case FieldHeight:
		return float64(img.Height), true
	default:
		return 0, false
	}
}

// Matches reports whether img satisfies this rule. An unknown field or
// operator never matches, so a malformed rule is inert rather than fatal.
func (r Rule) Matches(img Image) bool {
	actual, ok := r.actual(img)
	if !ok {
		return false
	}
	switch r.Op {
	case OpEqual:
		return within(actual, r.Value, r.Tolerance)
	case OpNotEqual:
		return !within(actual, r.Value, r.Tolerance)
	case OpLess:
		return actual < r.Value
	case OpLessEqual:
		return actual <= r.Value
	case OpGreater:
		return actual > r.Value
	case OpGreaterEqual:
		return actual >= r.Value
	default:
		return false
	}
}

// within reports whether actual is within a relative tolerance of target.
// A zero target falls back to an exact comparison, because a relative band
// around zero has no width.
func within(actual, target, tolerance float64) bool {
	if target == 0 {
		return actual == 0
	}
	return math.Abs(actual-target)/math.Abs(target) <= tolerance
}

// Describe renders the rule as one readable line for the interface.
func (r Rule) Describe() string {
	value := r.Label
	if value == "" {
		value = formatValue(r.Field, r.Value)
	}
	out := fmt.Sprintf("%s if %s %s %s", r.Action, r.Field, r.Op, value)
	if r.Tolerance > 0 && (r.Op == OpEqual || r.Op == OpNotEqual) {
		out += fmt.Sprintf(" (+/-%s)", FormatTolerance(r.Tolerance))
	}
	return out
}

// formatValue renders a bare rule value: ratios to four places, pixel counts
// as whole numbers.
func formatValue(field Field, v float64) string {
	if field == FieldAspect {
		return strconv.FormatFloat(v, 'f', 4, 64)
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// FormatTolerance renders a fractional tolerance as a percentage, for example
// 0.015 as "1.5%".
func FormatTolerance(t float64) string {
	return strings.TrimSuffix(strconv.FormatFloat(t*100, 'f', -1, 64), ".0") + "%"
}

// ParseRatio reads an aspect ratio written either as a pair, "16:9" or
// "16/10", or as a bare decimal such as "1.7778".
func ParseRatio(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("parse ratio: value is empty")
	}
	sep := strings.IndexAny(s, ":/")
	if sep < 0 {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, fmt.Errorf("parse ratio %q: %w", s, err)
		}
		if v <= 0 {
			return 0, fmt.Errorf("parse ratio %q: must be greater than zero", s)
		}
		return v, nil
	}
	w, err := strconv.ParseFloat(strings.TrimSpace(s[:sep]), 64)
	if err != nil {
		return 0, fmt.Errorf("parse ratio %q: bad width term: %w", s, err)
	}
	h, err := strconv.ParseFloat(strings.TrimSpace(s[sep+1:]), 64)
	if err != nil {
		return 0, fmt.Errorf("parse ratio %q: bad height term: %w", s, err)
	}
	if w <= 0 || h <= 0 {
		return 0, fmt.Errorf("parse ratio %q: both terms must be greater than zero", s)
	}
	return w / h, nil
}

// Policy is an ordered rule list with a fallback. The first rule that matches
// decides, which is what lets a narrow move rule sit above broad keep rules.
type Policy struct {
	Rules   []Rule `json:"rules"`
	Default Action `json:"default"`
}

// Decide returns the verdict for img and the index of the rule that produced
// it. The index is -1 when no rule matched and the default applied.
func (p Policy) Decide(img Image) (Action, int) {
	for i, r := range p.Rules {
		if r.Matches(img) {
			return r.Action, i
		}
	}
	return p.Default, -1
}

// DefaultPolicy is the shipped starting point: move anything 1080 pixels tall
// or shorter, keep 16:9 and 16:10 within one percent, and move the rest. It
// reproduces the wallpaper collection's own hand sort, which its test pins.
func DefaultPolicy() Policy {
	return Policy{
		Rules: []Rule{
			{Field: FieldHeight, Op: OpLessEqual, Value: 1080, Action: ActionMove},
			{Field: FieldAspect, Op: OpEqual, Value: 16.0 / 9.0, Tolerance: 0.01, Action: ActionKeep, Label: "16:9"},
			{Field: FieldAspect, Op: OpEqual, Value: 16.0 / 10.0, Tolerance: 0.01, Action: ActionKeep, Label: "16:10"},
		},
		Default: ActionMove,
	}
}

// Config is the complete saved state: where to read, where to move, what to
// consider an image, and the policy to apply.
type Config struct {
	SourceDir  string   `json:"source_dir"`
	DestDir    string   `json:"dest_dir"`
	Extensions []string `json:"extensions"`
	Policy     Policy   `json:"policy"`
}

// Validate reports the first reason this config cannot drive a sort. It checks
// shape only, never the filesystem, so it stays usable while the user is still
// typing a path.
func (c Config) Validate() error {
	switch {
	case strings.TrimSpace(c.SourceDir) == "":
		return fmt.Errorf("validate config: source directory is not set")
	case strings.TrimSpace(c.DestDir) == "":
		return fmt.Errorf("validate config: destination directory is not set")
	case c.SourceDir == c.DestDir:
		return fmt.Errorf("validate config: source and destination are the same directory")
	case len(c.Extensions) == 0:
		return fmt.Errorf("validate config: no file extensions are enabled")
	}
	return nil
}

// DefaultConfig is a new configuration with no directories chosen yet.
func DefaultConfig() Config {
	return Config{
		Extensions: []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".tif", ".tiff"},
		Policy:     DefaultPolicy(),
	}
}

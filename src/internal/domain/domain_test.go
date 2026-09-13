package domain

import "testing"

func TestImageAspect(t *testing.T) {
	tests := []struct {
		name string
		w, h int
		want float64
	}{
		{"16:9 UHD", 3840, 2160, 16.0 / 9.0},
		{"16:10 WUXGA", 1920, 1200, 1.6},
		{"square", 1000, 1000, 1.0},
		{"zero height guards", 1920, 0, 0},
		{"negative height guards", 1920, -1, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Image{Width: tc.w, Height: tc.h}.Aspect()
			if diff := got - tc.want; diff > 1e-9 || diff < -1e-9 {
				t.Errorf("Aspect() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParseRatio(t *testing.T) {
	tests := []struct {
		in      string
		want    float64
		wantErr bool
	}{
		{"16:9", 16.0 / 9.0, false},
		{"16/10", 1.6, false},
		{" 21 : 9 ", 21.0 / 9.0, false},
		{"1.7778", 1.7778, false},
		{"2", 2, false},
		{"16:0", 0, true},
		{"a:b", 0, true},
		{"", 0, true},
		{"-3:2", 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			got, err := ParseRatio(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseRatio(%q) = %v, want error", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseRatio(%q) returned error: %v", tc.in, err)
			}
			if diff := got - tc.want; diff > 1e-9 || diff < -1e-9 {
				t.Errorf("ParseRatio(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestRuleMatches(t *testing.T) {
	uhd := Image{Width: 3840, Height: 2160}   // 16:9, 1.7778
	odd := Image{Width: 2912, Height: 1632}   // 1.7843, 0.37% off 16:9
	wide := Image{Width: 3440, Height: 1440}  // 2.3889
	small := Image{Width: 1920, Height: 1080} // 16:9 but 1080 high

	tests := []struct {
		name string
		rule Rule
		img  Image
		want bool
	}{
		{"height lte matches at boundary", Rule{Field: FieldHeight, Op: OpLessEqual, Value: 1080}, small, true},
		{"height lte misses above boundary", Rule{Field: FieldHeight, Op: OpLessEqual, Value: 1080}, uhd, false},
		{"height lt misses at boundary", Rule{Field: FieldHeight, Op: OpLess, Value: 1080}, small, false},
		{"height lt matches below boundary", Rule{Field: FieldHeight, Op: OpLess, Value: 1080}, Image{Width: 1366, Height: 768}, true},
		{"width gte matches", Rule{Field: FieldWidth, Op: OpGreaterEqual, Value: 3840}, uhd, true},
		{"width gt misses at boundary", Rule{Field: FieldWidth, Op: OpGreater, Value: 3840}, uhd, false},
		{"aspect eq exact with zero tolerance", Rule{Field: FieldAspect, Op: OpEqual, Value: 16.0 / 9.0}, uhd, true},
		{"aspect eq rejects near miss at zero tolerance", Rule{Field: FieldAspect, Op: OpEqual, Value: 16.0 / 9.0}, odd, false},
		{"aspect eq accepts near miss within tolerance", Rule{Field: FieldAspect, Op: OpEqual, Value: 16.0 / 9.0, Tolerance: 0.01}, odd, true},
		{"aspect eq rejects outside tolerance", Rule{Field: FieldAspect, Op: OpEqual, Value: 16.0 / 9.0, Tolerance: 0.01}, wide, false},
		{"aspect ne inverts eq", Rule{Field: FieldAspect, Op: OpNotEqual, Value: 16.0 / 9.0, Tolerance: 0.01}, wide, true},
		{"aspect ne false inside tolerance", Rule{Field: FieldAspect, Op: OpNotEqual, Value: 16.0 / 9.0, Tolerance: 0.01}, odd, false},
		{"unknown field never matches", Rule{Field: "depth", Op: OpEqual, Value: 0}, uhd, false},
		{"unknown operator never matches", Rule{Field: FieldHeight, Op: "~=", Value: 2160}, uhd, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.rule.Matches(tc.img); got != tc.want {
				t.Errorf("Matches() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRuleMatchesZeroValueTolerance(t *testing.T) {
	// A zero target with a relative tolerance must not divide by zero.
	r := Rule{Field: FieldWidth, Op: OpEqual, Value: 0, Tolerance: 0.5}
	if r.Matches(Image{Width: 0, Height: 100}) != true {
		t.Error("zero equals zero should match")
	}
	if r.Matches(Image{Width: 10, Height: 100}) != false {
		t.Error("non-zero must not match a zero target")
	}
}

func TestPolicyDecideFirstMatchWins(t *testing.T) {
	p := DefaultPolicy()

	tests := []struct {
		name     string
		img      Image
		want     Action
		wantRule int
	}{
		{"1080-high 16:9 moves on the height rule", Image{Width: 1920, Height: 1080}, ActionMove, 0},
		{"UHD 16:9 keeps", Image{Width: 3840, Height: 2160}, ActionKeep, 1},
		{"WQXGA 16:10 keeps", Image{Width: 3840, Height: 2400}, ActionKeep, 2},
		{"near-16:9 inside tolerance keeps", Image{Width: 2912, Height: 1632}, ActionKeep, 1},
		{"ultrawide falls through to the default", Image{Width: 3440, Height: 1440}, ActionMove, -1},
		{"3:2 falls through to the default", Image{Width: 2560, Height: 1707}, ActionMove, -1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotAction, gotRule := p.Decide(tc.img)
			if gotAction != tc.want || gotRule != tc.wantRule {
				t.Errorf("Decide() = (%v, %d), want (%v, %d)", gotAction, gotRule, tc.want, tc.wantRule)
			}
		})
	}
}

func TestPolicyDecideEmptyUsesDefault(t *testing.T) {
	p := Policy{Default: ActionKeep}
	if action, idx := p.Decide(Image{Width: 100, Height: 100}); action != ActionKeep || idx != -1 {
		t.Errorf("Decide() = (%v, %d), want (keep, -1)", action, idx)
	}
}

// TestDefaultPolicyReproducesTheManualSort pins the default policy against the
// wallpaper collection's own 2025-12-17 hand sort, so a change to the shipped
// defaults that would re-file Chris's existing library fails here first.
func TestDefaultPolicyReproducesTheManualSort(t *testing.T) {
	moved := []Image{
		{Name: "adwaita-l.jpg", Width: 4096, Height: 4096},
		{Name: "the-sphere.png", Width: 3440, Height: 1440},
		{Name: "mountain.jpg", Width: 1920, Height: 1080},
		{Name: "groot_1.jpg", Width: 1366, Height: 768},
		{Name: "tree.jpg", Width: 1920, Height: 1280},
		{Name: "glowing-green-dew.jpg", Width: 6067, Height: 3467},
		{Name: "orange-art.jpg", Width: 3400, Height: 2024},
		{Name: "suzume1.jpg", Width: 8192, Height: 4320},
		{Name: "street1.jpg", Width: 3840, Height: 1875},
		{Name: "pexels-robert-clark.jpg", Width: 4936, Height: 3290},
	}
	kept := []Image{
		{Name: "3d-tech.jpg", Width: 3840, Height: 2160},
		{Name: "wallhaven3.jpg", Width: 1920, Height: 1200},
		{Name: "Fantasy-Landscape3.png", Width: 3831, Height: 2160},
		{Name: "golden-horizon.jpg", Width: 3200, Height: 1793},
		{Name: "comet.jpg", Width: 1920, Height: 1081},
		{Name: "anime3.png", Width: 5118, Height: 2878},
		{Name: "tron_legacy6.jpg", Width: 2880, Height: 1800},
	}

	p := DefaultPolicy()
	for _, img := range moved {
		if action, _ := p.Decide(img); action != ActionMove {
			t.Errorf("%s (%dx%d): got %v, want move", img.Name, img.Width, img.Height, action)
		}
	}
	for _, img := range kept {
		if action, _ := p.Decide(img); action != ActionKeep {
			t.Errorf("%s (%dx%d): got %v, want keep", img.Name, img.Width, img.Height, action)
		}
	}
}

func TestRuleDescribe(t *testing.T) {
	tests := []struct {
		rule Rule
		want string
	}{
		{Rule{Field: FieldHeight, Op: OpLessEqual, Value: 1080, Action: ActionMove}, "move if height <= 1080"},
		{Rule{Field: FieldAspect, Op: OpEqual, Value: 16.0 / 9.0, Tolerance: 0.01, Action: ActionKeep, Label: "16:9"}, "keep if aspect == 16:9 (+/-1%)"},
		{Rule{Field: FieldAspect, Op: OpGreater, Value: 2, Action: ActionMove}, "move if aspect > 2.0000"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.rule.Describe(); got != tc.want {
				t.Errorf("Describe() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	valid := Config{SourceDir: "/a", DestDir: "/a/sub", Extensions: []string{".jpg"}, Policy: DefaultPolicy()}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() on a valid config returned: %v", err)
	}

	tests := []struct {
		name string
		cfg  Config
	}{
		{"empty source", Config{DestDir: "/a/sub", Extensions: []string{".jpg"}}},
		{"empty destination", Config{SourceDir: "/a", Extensions: []string{".jpg"}}},
		{"identical directories", Config{SourceDir: "/a", DestDir: "/a", Extensions: []string{".jpg"}}},
		{"no extensions", Config{SourceDir: "/a", DestDir: "/a/sub"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.cfg.Validate(); err == nil {
				t.Error("Validate() = nil, want error")
			}
		})
	}
}

func TestDefaultConfigIsValid(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SourceDir = "/tmp/pics"
	cfg.DestDir = "/tmp/pics/rejects"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("DefaultConfig() is not usable: %v", err)
	}
	if len(cfg.Extensions) == 0 {
		t.Error("DefaultConfig() must ship a non-empty extension list")
	}
}

package sorter

import (
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rs/zerolog"

	"github.com/nerdexecutive/ne-image-sorter/internal/domain"
)

// fakeImages is an in-memory Images repository, so the service is tested
// without touching a filesystem and failure paths are reachable on demand.
type fakeImages struct {
	images   []domain.Image
	listErr  error
	moveErr  map[string]error
	moved    []string
	moveDest string
}

func (f *fakeImages) List(dir string, extensions []string) ([]domain.Image, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.images, nil
}

func (f *fakeImages) Move(img domain.Image, destDir string) (string, error) {
	if err, ok := f.moveErr[img.Name]; ok {
		return "", err
	}
	f.moved = append(f.moved, img.Name)
	f.moveDest = destDir
	return filepath.Join(destDir, img.Name), nil
}

func quietLogger() zerolog.Logger { return zerolog.New(io.Discard) }

func testConfig(images ...domain.Image) (domain.Config, *fakeImages) {
	cfg := domain.DefaultConfig()
	cfg.SourceDir = "/pics"
	cfg.DestDir = "/pics/rejects"
	return cfg, &fakeImages{images: images}
}

func TestPlanClassifiesEveryImage(t *testing.T) {
	cfg, repo := testConfig(
		domain.Image{Name: "uhd.jpg", Width: 3840, Height: 2160},       // keep, 16:9
		domain.Image{Name: "hd.jpg", Width: 1920, Height: 1080},        // move, height
		domain.Image{Name: "ultrawide.jpg", Width: 3440, Height: 1440}, // move, default
		domain.Image{Name: "wuxga.jpg", Width: 1920, Height: 1200},     // keep, 16:10
	)

	plan, err := New(repo, quietLogger()).Plan(cfg)
	if err != nil {
		t.Fatalf("Plan() returned error: %v", err)
	}
	if len(plan.Decisions) != 4 {
		t.Fatalf("Plan() produced %d decisions, want 4", len(plan.Decisions))
	}
	if plan.MoveCount != 2 || plan.KeepCount != 2 {
		t.Errorf("Plan() counts = %d move, %d keep; want 2 and 2", plan.MoveCount, plan.KeepCount)
	}

	want := map[string]domain.Action{
		"uhd.jpg": domain.ActionKeep, "hd.jpg": domain.ActionMove,
		"ultrawide.jpg": domain.ActionMove, "wuxga.jpg": domain.ActionKeep,
	}
	for _, d := range plan.Decisions {
		if d.Action != want[d.Image.Name] {
			t.Errorf("%s = %v, want %v", d.Image.Name, d.Action, want[d.Image.Name])
		}
		if d.Reason == "" {
			t.Errorf("%s has an empty reason", d.Image.Name)
		}
	}
}

func TestPlanReasonNamesTheDecidingRule(t *testing.T) {
	cfg, repo := testConfig(
		domain.Image{Name: "hd.jpg", Width: 1920, Height: 1080},
		domain.Image{Name: "ultrawide.jpg", Width: 3440, Height: 1440},
	)
	plan, err := New(repo, quietLogger()).Plan(cfg)
	if err != nil {
		t.Fatalf("Plan() returned error: %v", err)
	}

	byName := map[string]Decision{}
	for _, d := range plan.Decisions {
		byName[d.Image.Name] = d
	}
	if got := byName["hd.jpg"].Reason; !strings.Contains(got, "height") {
		t.Errorf("hd.jpg reason = %q, want it to name the height rule", got)
	}
	if got := byName["ultrawide.jpg"].Reason; !strings.Contains(strings.ToLower(got), "default") {
		t.Errorf("ultrawide.jpg reason = %q, want it to name the default", got)
	}
}

func TestPlanRejectsInvalidConfig(t *testing.T) {
	repo := &fakeImages{}
	if _, err := New(repo, quietLogger()).Plan(domain.Config{}); err == nil {
		t.Error("Plan() on an invalid config = nil error, want an error")
	}
}

func TestPlanWrapsListError(t *testing.T) {
	cfg, repo := testConfig()
	sentinel := errors.New("disk on fire")
	repo.listErr = sentinel

	_, err := New(repo, quietLogger()).Plan(cfg)
	if err == nil {
		t.Fatal("Plan() = nil error, want an error")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("Plan() error %v does not wrap the repository error", err)
	}
}

func TestPlanEmptySource(t *testing.T) {
	cfg, repo := testConfig()
	plan, err := New(repo, quietLogger()).Plan(cfg)
	if err != nil {
		t.Fatalf("Plan() returned error: %v", err)
	}
	if len(plan.Decisions) != 0 || plan.MoveCount != 0 {
		t.Error("Plan() on an empty source must produce an empty plan")
	}
}

func TestApplyMovesOnlyTheMoveDecisions(t *testing.T) {
	cfg, repo := testConfig(
		domain.Image{Name: "uhd.jpg", Width: 3840, Height: 2160},
		domain.Image{Name: "hd.jpg", Width: 1920, Height: 1080},
		domain.Image{Name: "ultrawide.jpg", Width: 3440, Height: 1440},
	)
	svc := New(repo, quietLogger())
	plan, err := svc.Plan(cfg)
	if err != nil {
		t.Fatalf("Plan() returned error: %v", err)
	}

	report, err := svc.Apply(cfg, plan)
	if err != nil {
		t.Fatalf("Apply() returned error: %v", err)
	}
	if report.Moved != 2 || report.Failed != 0 {
		t.Errorf("Apply() = %d moved, %d failed; want 2 and 0", report.Moved, report.Failed)
	}
	if len(repo.moved) != 2 {
		t.Fatalf("repository saw %d moves, want 2", len(repo.moved))
	}
	for _, name := range repo.moved {
		if name == "uhd.jpg" {
			t.Error("Apply() moved a keep decision")
		}
	}
	if repo.moveDest != cfg.DestDir {
		t.Errorf("Apply() moved into %q, want %q", repo.moveDest, cfg.DestDir)
	}
}

func TestApplyContinuesPastAFailureAndReportsIt(t *testing.T) {
	cfg, repo := testConfig(
		domain.Image{Name: "a.jpg", Width: 3440, Height: 1440},
		domain.Image{Name: "b.jpg", Width: 3440, Height: 1440},
		domain.Image{Name: "c.jpg", Width: 3440, Height: 1440},
	)
	repo.moveErr = map[string]error{"b.jpg": errors.New("permission denied")}

	svc := New(repo, quietLogger())
	plan, err := svc.Plan(cfg)
	if err != nil {
		t.Fatalf("Plan() returned error: %v", err)
	}

	report, err := svc.Apply(cfg, plan)
	if err != nil {
		t.Fatalf("Apply() must not abort on a single failure, got: %v", err)
	}
	if report.Moved != 2 || report.Failed != 1 {
		t.Errorf("Apply() = %d moved, %d failed; want 2 and 1", report.Moved, report.Failed)
	}
	if len(report.Errors) != 1 || !strings.Contains(report.Errors[0], "b.jpg") {
		t.Errorf("Apply() errors = %v, want one naming b.jpg", report.Errors)
	}
}

func TestApplyRejectsInvalidConfig(t *testing.T) {
	repo := &fakeImages{}
	if _, err := New(repo, quietLogger()).Apply(domain.Config{}, Plan{}); err == nil {
		t.Error("Apply() on an invalid config = nil error, want an error")
	}
}

func TestApplyEmptyPlanIsANoOp(t *testing.T) {
	cfg, repo := testConfig()
	report, err := New(repo, quietLogger()).Apply(cfg, Plan{})
	if err != nil {
		t.Fatalf("Apply() returned error: %v", err)
	}
	if report.Moved != 0 || report.Failed != 0 || len(repo.moved) != 0 {
		t.Error("Apply() on an empty plan must move nothing")
	}
}

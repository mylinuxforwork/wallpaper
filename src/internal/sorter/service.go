// Package sorter applies a policy to a directory of images. It plans first
// and moves second, so the interface can show the user exactly what will
// happen before anything on disk changes.
package sorter

import (
	"fmt"

	"github.com/rs/zerolog"

	"github.com/nerdexecutive/ne-image-sorter/internal/domain"
	"github.com/nerdexecutive/ne-image-sorter/internal/repository"
)

// Decision is one image and the verdict the policy reached for it.
type Decision struct {
	Image  domain.Image
	Action domain.Action
	Reason string
}

// Plan is a complete classification of a source directory. Nothing has moved.
type Plan struct {
	Decisions []Decision
	MoveCount int
	KeepCount int
}

// Moves returns just the decisions that relocate a file.
func (p Plan) Moves() []Decision {
	out := make([]Decision, 0, p.MoveCount)
	for _, d := range p.Decisions {
		if d.Action == domain.ActionMove {
			out = append(out, d)
		}
	}
	return out
}

// Report is the outcome of applying a plan.
type Report struct {
	Moved  int
	Failed int
	Errors []string
}

// Service plans and applies sorts against an image repository.
type Service struct {
	images repository.Images
	log    zerolog.Logger
}

// New returns a Service reading and writing through images.
func New(images repository.Images, log zerolog.Logger) *Service {
	return &Service{images: images, log: log}
}

// Plan classifies every image in the configured source directory without
// changing anything on disk.
func (s *Service) Plan(cfg domain.Config) (Plan, error) {
	if err := cfg.Validate(); err != nil {
		return Plan{}, fmt.Errorf("plan sort: %w", err)
	}

	images, err := s.images.List(cfg.SourceDir, cfg.Extensions)
	if err != nil {
		return Plan{}, fmt.Errorf("plan sort: %w", err)
	}

	plan := Plan{Decisions: make([]Decision, 0, len(images))}
	for _, img := range images {
		action, ruleIdx := cfg.Policy.Decide(img)
		plan.Decisions = append(plan.Decisions, Decision{
			Image:  img,
			Action: action,
			Reason: reason(cfg.Policy, ruleIdx),
		})
		if action == domain.ActionMove {
			plan.MoveCount++
		} else {
			plan.KeepCount++
		}
	}

	s.log.Info().
		Str("source", cfg.SourceDir).
		Int("scanned", len(images)).
		Int("move", plan.MoveCount).
		Int("keep", plan.KeepCount).
		Msg("planned sort")
	return plan, nil
}

// reason renders why a decision was reached, naming the deciding rule or the
// policy default.
func reason(p domain.Policy, ruleIdx int) string {
	if ruleIdx < 0 || ruleIdx >= len(p.Rules) {
		return fmt.Sprintf("no rule matched, default is %s", p.Default)
	}
	return fmt.Sprintf("rule %d: %s", ruleIdx+1, p.Rules[ruleIdx].Describe())
}

// Apply performs the move decisions in plan. A single file that cannot be
// moved is recorded and the run continues, because one locked file must not
// abandon the rest of a sort part-done.
func (s *Service) Apply(cfg domain.Config, plan Plan) (Report, error) {
	if err := cfg.Validate(); err != nil {
		return Report{}, fmt.Errorf("apply sort: %w", err)
	}

	var report Report
	for _, d := range plan.Moves() {
		dest, err := s.images.Move(d.Image, cfg.DestDir)
		if err != nil {
			report.Failed++
			report.Errors = append(report.Errors, fmt.Sprintf("%s: %v", d.Image.Name, err))
			s.log.Error().Err(err).Str("image", d.Image.Name).Msg("move failed")
			continue
		}
		report.Moved++
		s.log.Info().
			Str("image", d.Image.Name).
			Str("resolution", d.Image.Resolution()).
			Str("dest", dest).
			Str("reason", d.Reason).
			Msg("moved")
	}

	s.log.Info().Int("moved", report.Moved).Int("failed", report.Failed).Msg("applied sort")
	return report, nil
}

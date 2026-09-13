package main

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rs/zerolog"

	"github.com/nerdexecutive/ne-image-sorter/internal/domain"
	"github.com/nerdexecutive/ne-image-sorter/internal/repository"
	"github.com/nerdexecutive/ne-image-sorter/internal/sorter"
)

// writeJPEG renders a w-by-h JPEG at path.
func writeJPEG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{R: 1, G: 2, B: 3, A: 255})
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// fixture builds a source directory holding one 16:9 image that stays and one
// ultrawide image that moves, and returns a matching config.
func fixture(t *testing.T) domain.Config {
	t.Helper()
	src := t.TempDir()
	writeJPEG(t, filepath.Join(src, "keep-16x9.jpg"), 1920, 1088) // 16:9, over 1080 tall
	writeJPEG(t, filepath.Join(src, "move-wide.jpg"), 3440, 1440) // 2.39:1

	cfg := domain.DefaultConfig()
	cfg.SourceDir = src
	cfg.DestDir = filepath.Join(src, "rejects")
	return cfg
}

func newService() *sorter.Service {
	return sorter.New(repository.NewFileImages(), zerolog.New(io.Discard))
}

func TestSortHeadlessMovesAndReports(t *testing.T) {
	cfg := fixture(t)
	var out bytes.Buffer

	if err := sortHeadless(&out, newService(), cfg, false); err != nil {
		t.Fatalf("sortHeadless returned error: %v", err)
	}

	got := out.String()
	for _, want := range []string{"move-wide.jpg", "1 to move", "1 to keep", "2 scanned", "moved 1, failed 0"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q, got:\n%s", want, got)
		}
	}
	if strings.Contains(got, "move  keep-16x9.jpg") {
		t.Error("the keep decision must not be listed as a move")
	}

	if _, err := os.Stat(filepath.Join(cfg.DestDir, "move-wide.jpg")); err != nil {
		t.Errorf("the moved file is not in the destination: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cfg.SourceDir, "keep-16x9.jpg")); err != nil {
		t.Errorf("the kept file left the source: %v", err)
	}
}

func TestSortHeadlessDryRunMovesNothing(t *testing.T) {
	cfg := fixture(t)
	var out bytes.Buffer

	if err := sortHeadless(&out, newService(), cfg, true); err != nil {
		t.Fatalf("sortHeadless returned error: %v", err)
	}
	if !strings.Contains(out.String(), "dry run: nothing was moved") {
		t.Error("a dry run must say so")
	}
	if _, err := os.Stat(filepath.Join(cfg.SourceDir, "move-wide.jpg")); err != nil {
		t.Error("a dry run moved a file")
	}
	if _, err := os.Stat(cfg.DestDir); !os.IsNotExist(err) {
		t.Error("a dry run created the destination directory")
	}
}

func TestSortHeadlessRejectsAnInvalidConfig(t *testing.T) {
	var out bytes.Buffer
	err := sortHeadless(&out, newService(), domain.Config{}, false)
	if err == nil {
		t.Fatal("sortHeadless on an empty config = nil error, want an error")
	}
	if !strings.Contains(err.Error(), "headless sort") {
		t.Errorf("error %q is not wrapped with its operation", err)
	}
}

func TestSortHeadlessMissingSourceDirectory(t *testing.T) {
	cfg := domain.DefaultConfig()
	cfg.SourceDir = filepath.Join(t.TempDir(), "absent")
	cfg.DestDir = filepath.Join(t.TempDir(), "rejects")

	var out bytes.Buffer
	if err := sortHeadless(&out, newService(), cfg, false); err == nil {
		t.Error("a missing source directory must be reported, not silently empty")
	}
}

func TestDefaultPathsAreAbsoluteOrLocal(t *testing.T) {
	if p := defaultConfigPath(); p == "" {
		t.Error("defaultConfigPath returned an empty path")
	}
	if p := defaultLogDir(); p == "" {
		t.Error("defaultLogDir returned an empty path")
	}
}

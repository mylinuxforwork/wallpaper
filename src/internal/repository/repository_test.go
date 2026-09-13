package repository

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/nerdexecutive/ne-image-sorter/internal/domain"
)

// writeImage renders a w-by-h image at path in the format implied by its
// extension, so the dimension reader is exercised against real file headers.
func writeImage(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{R: 1, G: 2, B: 3, A: 255})

	var buf bytes.Buffer
	var err error
	switch filepath.Ext(path) {
	case ".png":
		err = png.Encode(&buf, img)
	case ".gif":
		err = gif.Encode(&buf, img, nil)
	default:
		err = jpeg.Encode(&buf, img, nil)
	}
	if err != nil {
		t.Fatalf("encode %s: %v", path, err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestFileImagesListReadsDimensions(t *testing.T) {
	dir := t.TempDir()
	writeImage(t, filepath.Join(dir, "uhd.jpg"), 384, 216)
	writeImage(t, filepath.Join(dir, "tall.png"), 100, 200)
	writeImage(t, filepath.Join(dir, "anim.gif"), 64, 64)

	got, err := NewFileImages().List(dir, []string{".jpg", ".png", ".gif"})
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("List() returned %d images, want 3", len(got))
	}

	want := map[string][2]int{"uhd.jpg": {384, 216}, "tall.png": {100, 200}, "anim.gif": {64, 64}}
	for _, img := range got {
		w, ok := want[img.Name]
		if !ok {
			t.Errorf("unexpected image %q", img.Name)
			continue
		}
		if img.Width != w[0] || img.Height != w[1] {
			t.Errorf("%s = %dx%d, want %dx%d", img.Name, img.Width, img.Height, w[0], w[1])
		}
		if img.Path != filepath.Join(dir, img.Name) {
			t.Errorf("%s path = %q, want an absolute path under the source", img.Name, img.Path)
		}
	}
}

func TestFileImagesListIsSortedByName(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"c.jpg", "a.jpg", "b.jpg"} {
		writeImage(t, filepath.Join(dir, n), 10, 10)
	}
	got, err := NewFileImages().List(dir, []string{".jpg"})
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}
	for i, want := range []string{"a.jpg", "b.jpg", "c.jpg"} {
		if got[i].Name != want {
			t.Errorf("List()[%d] = %q, want %q", i, got[i].Name, want)
		}
	}
}

func TestFileImagesListFiltersAndSkips(t *testing.T) {
	dir := t.TempDir()
	writeImage(t, filepath.Join(dir, "keep.jpg"), 10, 10)
	writeImage(t, filepath.Join(dir, "wrong-ext.png"), 10, 10)
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A file with an enabled extension whose bytes are not an image must be
	// skipped rather than aborting the scan.
	if err := os.WriteFile(filepath.Join(dir, "broken.jpg"), []byte("not an image"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub.jpg"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := NewFileImages().List(dir, []string{".jpg"})
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "keep.jpg" {
		t.Fatalf("List() = %+v, want only keep.jpg", got)
	}
}

func TestFileImagesListExtensionMatchIsCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	writeImage(t, filepath.Join(dir, "shouty.JPG"), 10, 10)
	got, err := NewFileImages().List(dir, []string{".jpg"})
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("List() returned %d images, want 1", len(got))
	}
}

func TestFileImagesListMissingDirectory(t *testing.T) {
	if _, err := NewFileImages().List(filepath.Join(t.TempDir(), "absent"), []string{".jpg"}); err == nil {
		t.Error("List() on a missing directory = nil error, want an error")
	}
}

func TestFileImagesMoveCreatesDestination(t *testing.T) {
	src, dst := t.TempDir(), filepath.Join(t.TempDir(), "rejects")
	writeImage(t, filepath.Join(src, "a.jpg"), 10, 10)

	img := domain.Image{Path: filepath.Join(src, "a.jpg"), Name: "a.jpg"}
	got, err := NewFileImages().Move(img, dst)
	if err != nil {
		t.Fatalf("Move() returned error: %v", err)
	}
	if want := filepath.Join(dst, "a.jpg"); got != want {
		t.Errorf("Move() = %q, want %q", got, want)
	}
	if _, err := os.Stat(got); err != nil {
		t.Errorf("destination file missing: %v", err)
	}
	if _, err := os.Stat(img.Path); !os.IsNotExist(err) {
		t.Error("source file still present after Move()")
	}
}

func TestFileImagesMoveDoesNotOverwrite(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	writeImage(t, filepath.Join(src, "a.jpg"), 10, 10)
	writeImage(t, filepath.Join(dst, "a.jpg"), 99, 99)

	img := domain.Image{Path: filepath.Join(src, "a.jpg"), Name: "a.jpg"}
	got, err := NewFileImages().Move(img, dst)
	if err != nil {
		t.Fatalf("Move() returned error: %v", err)
	}
	if want := filepath.Join(dst, "a-1.jpg"); got != want {
		t.Fatalf("Move() = %q, want %q", got, want)
	}
	// The pre-existing file must be untouched at its original dimensions.
	existing, err := NewFileImages().List(dst, []string{".jpg"})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range existing {
		if e.Name == "a.jpg" && (e.Width != 99 || e.Height != 99) {
			t.Error("Move() overwrote the pre-existing destination file")
		}
	}
}

func TestFileImagesMoveMissingSource(t *testing.T) {
	img := domain.Image{Path: filepath.Join(t.TempDir(), "gone.jpg"), Name: "gone.jpg"}
	if _, err := NewFileImages().Move(img, t.TempDir()); err == nil {
		t.Error("Move() on a missing source = nil error, want an error")
	}
}

func TestJSONConfigRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	repo := NewJSONConfig(path)

	want := domain.DefaultConfig()
	want.SourceDir = "/pics"
	want.DestDir = "/pics/rejects"
	if err := repo.Save(want); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	got, err := repo.Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if got.SourceDir != want.SourceDir || got.DestDir != want.DestDir {
		t.Errorf("Load() directories = %q, %q", got.SourceDir, got.DestDir)
	}
	if len(got.Policy.Rules) != len(want.Policy.Rules) {
		t.Fatalf("Load() returned %d rules, want %d", len(got.Policy.Rules), len(want.Policy.Rules))
	}
	for i, r := range got.Policy.Rules {
		if r != want.Policy.Rules[i] {
			t.Errorf("rule %d = %+v, want %+v", i, r, want.Policy.Rules[i])
		}
	}
	if got.Policy.Default != want.Policy.Default {
		t.Errorf("Load() default = %q, want %q", got.Policy.Default, want.Policy.Default)
	}
}

func TestJSONConfigLoadMissingFileReturnsDefaults(t *testing.T) {
	repo := NewJSONConfig(filepath.Join(t.TempDir(), "absent.json"))
	got, err := repo.Load()
	if err != nil {
		t.Fatalf("Load() on a missing file returned error: %v", err)
	}
	if len(got.Policy.Rules) != len(domain.DefaultPolicy().Rules) {
		t.Error("Load() on a missing file must return the shipped defaults")
	}
}

func TestJSONConfigLoadCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := NewJSONConfig(path).Load(); err == nil {
		t.Error("Load() on a corrupt file = nil error, want an error")
	}
}

// Compile-time proof that both concrete types satisfy the interfaces the
// service depends on.
var (
	_ Images = (*FileImages)(nil)
	_ Config = (*JSONConfig)(nil)
)

package repository

import (
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	// Registered for their image.DecodeConfig side effect, which reads
	// dimensions from the file header without decoding the pixels.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"

	"github.com/nerdexecutive/ne-image-sorter/internal/domain"
)

// FileImages is the filesystem-backed Images repository.
type FileImages struct{}

// NewFileImages returns an Images repository reading the local filesystem.
func NewFileImages() *FileImages { return &FileImages{} }

// List implements Images.
func (r *FileImages) List(dir string, extensions []string) ([]domain.Image, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("list images in %s: %w", dir, err)
	}

	enabled := make(map[string]struct{}, len(extensions))
	for _, e := range extensions {
		enabled[strings.ToLower(e)] = struct{}{}
	}

	var out []domain.Image
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if _, ok := enabled[strings.ToLower(filepath.Ext(name))]; !ok {
			continue
		}
		path := filepath.Join(dir, name)
		w, h, err := readDimensions(path)
		if err != nil {
			// An unreadable or non-image file is not a scan failure. Skipping
			// it keeps one bad file from stopping a sort of hundreds.
			continue
		}
		out = append(out, domain.Image{Path: path, Name: name, Width: w, Height: h})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// readDimensions reads an image's pixel dimensions from its header.
func readDimensions(path string) (int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, fmt.Errorf("decode header of %s: %w", path, err)
	}
	return cfg.Width, cfg.Height, nil
}

// Move implements Images.
func (r *FileImages) Move(img domain.Image, destDir string) (string, error) {
	if _, err := os.Stat(img.Path); err != nil {
		return "", fmt.Errorf("move %s: source unreadable: %w", img.Name, err)
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("move %s: create %s: %w", img.Name, destDir, err)
	}

	target, err := freeName(destDir, img.Name)
	if err != nil {
		return "", fmt.Errorf("move %s: %w", img.Name, err)
	}
	if err := os.Rename(img.Path, target); err != nil {
		// Rename fails across filesystems, so fall back to a copy.
		if err := copyFile(img.Path, target); err != nil {
			return "", fmt.Errorf("move %s: %w", img.Name, err)
		}
		if err := os.Remove(img.Path); err != nil {
			return "", fmt.Errorf("move %s: remove source after copy: %w", img.Name, err)
		}
	}
	return target, nil
}

// freeName returns a path in dir that no file occupies, suffixing the base
// name with -1, -2 and so on until one is free.
func freeName(dir, name string) (string, error) {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 0; i < 1000; i++ {
		candidate := filepath.Join(dir, name)
		if i > 0 {
			candidate = filepath.Join(dir, fmt.Sprintf("%s-%d%s", base, i, ext))
		}
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no free name for %s in %s after 1000 attempts", name, dir)
}

// copyFile writes src to dst, which must not already exist.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("create destination %s: %w", dst, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return fmt.Errorf("copy to %s: %w", dst, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close destination %s: %w", dst, err)
	}
	return nil
}

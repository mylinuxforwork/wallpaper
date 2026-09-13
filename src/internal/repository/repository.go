// Package repository isolates every filesystem concern behind two narrow
// interfaces, so the sorting service can be driven against fakes and the
// concrete implementations can be swapped without touching the policy code.
package repository

import "github.com/nerdexecutive/ne-image-sorter/internal/domain"

// Images is the collection of image files the sorter reads and relocates.
type Images interface {
	// List returns every file in dir whose extension is enabled, with its
	// pixel dimensions read from the file header. Files that are not
	// decodable images are skipped rather than reported as errors.
	List(dir string, extensions []string) ([]domain.Image, error)

	// Move relocates img into destDir, creating the directory if needed, and
	// returns the path it was written to. An existing file at the target name
	// is never overwritten; the moved file is suffixed instead.
	Move(img domain.Image, destDir string) (string, error)
}

// Config is the persisted configuration.
type Config interface {
	// Load returns the stored configuration, or the shipped defaults when
	// nothing has been saved yet.
	Load() (domain.Config, error)

	// Save writes the configuration, creating the parent directory if needed.
	Save(cfg domain.Config) error
}

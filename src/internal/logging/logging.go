// Package logging opens the application's log file. Output goes to a file and
// never to the console: [Rule: Log to Files] requires it, and a terminal UI
// redraws the screen continuously, so a stray console write corrupts the
// display rather than informing anyone.
package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
)

// Open creates dir if needed and returns a logger writing to
// ne-image-sorter.log inside it, plus the file to close on shutdown.
func Open(dir string) (zerolog.Logger, *os.File, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return zerolog.Nop(), nil, fmt.Errorf("open log directory %s: %w", dir, err)
	}

	path := filepath.Join(dir, "ne-image-sorter.log")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return zerolog.Nop(), nil, fmt.Errorf("open log file %s: %w", path, err)
	}

	log := zerolog.New(f).With().Timestamp().Logger()
	log.Info().Str("version", "dev").Time("started", time.Now()).Msg("ne-image-sorter started")
	return log, f, nil
}

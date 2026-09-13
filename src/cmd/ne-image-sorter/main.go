// Command ne-image-sorter sorts a directory of images by aspect ratio and
// resolution, moving everything a configurable policy rejects into a second
// directory. It runs as a terminal interface by default, and headless with
// --sort for scripted use.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nerdexecutive/ne-image-sorter/internal/domain"
	"github.com/nerdexecutive/ne-image-sorter/internal/logging"
	"github.com/nerdexecutive/ne-image-sorter/internal/repository"
	"github.com/nerdexecutive/ne-image-sorter/internal/sorter"
	"github.com/nerdexecutive/ne-image-sorter/internal/tui"
)

// version is overridden at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if err := run(os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "ne-image-sorter:", err)
		os.Exit(1)
	}
}

// run wires the application together. Result output goes to out rather than
// to the process stdout directly, which keeps the headless path testable and
// leaves the log file as the only place this tool writes diagnostics.
func run(out io.Writer) error {
	var (
		showVersion = flag.Bool("version", false, "print the version and exit")
		configPath  = flag.String("config", defaultConfigPath(), "path to the configuration file")
		logDir      = flag.String("log-dir", defaultLogDir(), "directory to write the log file into")
		source      = flag.String("source", "", "override the configured source directory")
		dest        = flag.String("dest", "", "override the configured destination directory")
		dryRun      = flag.Bool("dry-run", false, "with --sort, report what would move without moving it")
		headless    = flag.Bool("sort", false, "sort without the interface and exit")
	)
	flag.Parse()

	if *showVersion {
		fmt.Fprintln(out, "ne-image-sorter", version)
		return nil
	}

	log, logFile, err := logging.Open(*logDir)
	if err != nil {
		return fmt.Errorf("start ne-image-sorter: %w", err)
	}
	defer logFile.Close()

	configs := repository.NewJSONConfig(*configPath)
	cfg, err := configs.Load()
	if err != nil {
		return fmt.Errorf("start ne-image-sorter: %w", err)
	}
	if *source != "" {
		cfg.SourceDir = *source
	}
	if *dest != "" {
		cfg.DestDir = *dest
	}

	svc := sorter.New(repository.NewFileImages(), log)
	if *headless {
		return sortHeadless(out, svc, cfg, *dryRun)
	}

	program := tea.NewProgram(tui.NewApp(cfg, configs, svc, log), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		log.Error().Err(err).Msg("interface exited with an error")
		return fmt.Errorf("run interface: %w", err)
	}
	return nil
}

// sortHeadless runs one sort without the interface, for scripts and cron.
func sortHeadless(out io.Writer, svc *sorter.Service, cfg domain.Config, dryRun bool) error {
	plan, err := svc.Plan(cfg)
	if err != nil {
		return fmt.Errorf("headless sort: %w", err)
	}

	for _, d := range plan.Moves() {
		fmt.Fprintf(out, "move  %-40s %-11s %s\n", d.Image.Name, d.Image.Resolution(), d.Reason)
	}
	fmt.Fprintf(out, "\n%d to move, %d to keep, %d scanned\n",
		plan.MoveCount, plan.KeepCount, len(plan.Decisions))

	if dryRun {
		fmt.Fprintln(out, "dry run: nothing was moved")
		return nil
	}

	report, err := svc.Apply(cfg, plan)
	if err != nil {
		return fmt.Errorf("headless sort: %w", err)
	}
	fmt.Fprintf(out, "moved %d, failed %d\n", report.Moved, report.Failed)
	for _, e := range report.Errors {
		fmt.Fprintln(out, "  failed:", e)
	}
	if report.Failed > 0 {
		return fmt.Errorf("headless sort: %d image(s) could not be moved", report.Failed)
	}
	return nil
}

// defaultConfigPath is the per-user configuration location, falling back to
// the working directory when the user config directory is unavailable.
func defaultConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "ne-image-sorter.json"
	}
	return filepath.Join(dir, "ne-image-sorter", "config.json")
}

// defaultLogDir is the per-user log location, falling back to a local logs
// directory.
func defaultLogDir() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "logs"
	}
	return filepath.Join(dir, "ne-image-sorter")
}

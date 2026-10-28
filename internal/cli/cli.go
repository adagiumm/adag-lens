// Package cli connects command-line arguments to the scanner and the
// report renderers. It owns flag parsing and exit codes.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/adagiumm/adag-lens/internal/report"
	"github.com/adagiumm/adag-lens/internal/scan"
	"github.com/adagiumm/adag-lens/internal/version"
)

// Exit codes. Warnings found in a repository are not errors: they are
// results of a successful scan, so they get a different code than failures.
const (
	exitOK       = 0 // scan completed, no warnings
	exitWarnings = 1 // scan completed, warnings found
	exitError    = 2 // invalid arguments or fatal error
)

const usageText = `adag-lens - a quick health check for a repository

Usage:
  adag-lens [flags] [path]

If path is omitted, the current directory is used.
Flags must come before the path.

Flags:
  --format string        output format: text or json (default "text")
  --max-file-size size   report files larger than this (default 10MiB)
                         units: B, KiB, MiB, GiB (binary; KB and MB are not accepted)
  --exclude-dir name     skip directories with this name at any depth; repeatable
                         (added to the defaults: .git, node_modules, vendor,
                         target, dist, build)
  --version              print the version and exit
  --help                 show this help

Exclusions match directory names only. .gitignore files are not read.

Exit codes:
  0  scan completed, no warnings
  1  scan completed, warnings found
  2  invalid arguments or fatal error
`

type options struct {
	format      string
	path        string
	scan        scan.Options
	showVersion bool
}

// Run executes adag-lens and returns the process exit code. Output streams
// are parameters (instead of os.Stdout) so tests can capture them.
func Run(args []string, stdout, stderr io.Writer) int {
	opts, err := parseArgs(args)
	if errors.Is(err, flag.ErrHelp) {
		fmt.Fprint(stdout, usageText)
		return exitOK
	}
	if err != nil {
		fmt.Fprintf(stderr, "adag-lens: %v\nRun 'adag-lens --help' for usage.\n", err)
		return exitError
	}

	if opts.showVersion {
		fmt.Fprintf(stdout, "adag-lens %s\n", version.String())
		return exitOK
	}

	root, err := resolveDir(opts.path)
	if err != nil {
		fmt.Fprintf(stderr, "adag-lens: %v\n", err)
		return exitError
	}

	start := time.Now()
	result, err := scan.Run(root, opts.scan)
	if err != nil {
		fmt.Fprintf(stderr, "adag-lens: %v\n", err)
		return exitError
	}
	elapsed := time.Since(start)

	if opts.format == "json" {
		err = report.WriteJSON(stdout, result)
	} else {
		err = report.WriteText(stdout, result, report.TextOptions{
			Color:    useColor(stdout),
			Duration: elapsed,
		})
	}
	if err != nil {
		fmt.Fprintf(stderr, "adag-lens: writing report: %v\n", err)
		return exitError
	}

	if result.Summary.Warn > 0 {
		return exitWarnings
	}
	return exitOK
}

func parseArgs(args []string) (options, error) {
	flags := flag.NewFlagSet("adag-lens", flag.ContinueOnError)
	// We print our own errors and usage, so silence the package's defaults.
	flags.SetOutput(io.Discard)

	format := flags.String("format", "text", "output format: text or json")

	showVersion := flags.Bool("version", false, "print the version and exit")

	// flag.Func calls these functions each time the flag appears, which
	// makes --exclude-dir repeatable. An error returned here is reported by
	// the flag package as: invalid value "..." for flag -name: <error>.
	// scanOpts starts from the defaults, so single source of truth lives in
	// the scan package.
	scanOpts := scan.DefaultOptions()
	flags.Func("max-file-size", "report files larger than this size", func(s string) error {
		n, err := scan.ParseSize(s)
		if err != nil {
			return err
		}
		scanOpts.MaxFileSize = n
		return nil
	})
	flags.Func("exclude-dir", "directory name to skip (repeatable)", func(s string) error {
		if err := validateDirName(s); err != nil {
			return err
		}
		scanOpts.ExcludeDirs = append(scanOpts.ExcludeDirs, s)
		return nil
	})

	if err := flags.Parse(args); err != nil {
		return options{}, err
	}

	if *format != "text" && *format != "json" {
		return options{}, fmt.Errorf("unsupported format %q (use text or json)", *format)
	}

	path := "."
	switch flags.NArg() {
	case 0:
	case 1:
		path = flags.Arg(0)
	default:
		return options{}, fmt.Errorf("expected at most one path, got %d", flags.NArg())
	}

	return options{format: *format, path: path, scan: scanOpts, showVersion: *showVersion}, nil
}

// validateDirName rejects values that could never match. Exclusions compare
// single directory names, so "src/generated" would silently do nothing.
func validateDirName(name string) error {
	if name == "" {
		return errors.New("directory name must not be empty")
	}
	if strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("%q looks like a path; give a directory name only (it is matched at any depth)", name)
	}
	return nil
}

// resolveDir turns the user's path into an absolute directory path, or
// returns an error that explains what is wrong in plain language.
func resolveDir(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("cannot resolve %q: %w", path, err)
	}

	// Stat follows symlinks. That is fine for the root: the user named this
	// path explicitly. (The scanner will treat symlinks differently inside.)
	info, err := os.Stat(abs)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "", fmt.Errorf("path does not exist: %s", path)
	case errors.Is(err, fs.ErrPermission):
		return "", fmt.Errorf("permission denied: %s", path)
	case err != nil:
		return "", fmt.Errorf("cannot access %s: %w", path, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", path)
	}
	return abs, nil
}

// useColor reports whether w should receive ANSI colors. Color is off when
// NO_COLOR is set (https://no-color.org), when TERM=dumb, when w is not an
// interactive terminal (piped or redirected output), and on Windows, where
// older consoles print escape codes literally unless ANSI mode is enabled.
func useColor(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	if runtime.GOOS == "windows" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

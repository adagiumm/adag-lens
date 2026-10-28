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
	"time"

	"github.com/adagiumm/adag-lens/internal/report"
	"github.com/adagiumm/adag-lens/internal/scan"
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
  --format string   output format: text or json (default "text")
  --help            show this help

Exit codes:
  0  scan completed, no warnings
  1  scan completed, warnings found
  2  invalid arguments or fatal error
`

type options struct {
	format string
	path   string
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

	root, err := resolveDir(opts.path)
	if err != nil {
		fmt.Fprintf(stderr, "adag-lens: %v\n", err)
		return exitError
	}

	start := time.Now()
	result, err := scan.Run(root, scan.DefaultOptions())
	if err != nil {
		fmt.Fprintf(stderr, "adag-lens: %v\n", err)
		return exitError
	}
	elapsed := time.Since(start)

	// TEMPORARY: until Step 5 adds JSON, --format json prints the text report.
	err = report.WriteText(stdout, result, report.TextOptions{
		Color:    useColor(stdout),
		Duration: elapsed,
	})
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

	return options{format: *format, path: path}, nil
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

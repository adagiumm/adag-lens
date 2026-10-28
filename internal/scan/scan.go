// Package scan inspects a repository directory and produces a Result.
// It performs no printing: rendering is the report package's job.
package scan

import (
	"errors"
	"fmt"
	"os"
)

// Status is the outcome of a single check.
type Status string

const (
	StatusPass Status = "pass"
	StatusWarn Status = "warn"
	StatusInfo Status = "info"
)

// Check is the outcome of one individual check.
type Check struct {
	ID      string // stable identifier, e.g. "readme"
	Status  Status
	Message string   // short, human-readable finding
	Details []string // optional extra lines, e.g. file names
}

// Summary counts checks by status.
type Summary struct {
	Pass int
	Warn int
	Info int
}

// LargeFile is a file whose size exceeds the configured threshold.
type LargeFile struct {
	Path string // relative to the scanned directory, always with forward slashes
	Size int64  // bytes
}

// Result is everything a scan learned about a repository.
type Result struct {
	Path         string      // directory that was scanned
	GitRoot      string      // top of the Git working tree; empty if none found
	ProjectTypes []string    // detected project types, in a fixed order
	Checks       []Check     // always in the same order
	LargeFiles   []LargeFile // biggest first
	Unreadable   []string    // paths the scan could not read, relative to Path
	Summary      Summary
}

// Options configures a scan. Use DefaultOptions as the starting point.
type Options struct {
	// MaxFileSize is the threshold in bytes: files strictly larger than
	// this are reported. It must be positive.
	MaxFileSize int64

	// ExcludeDirs lists directory names that are skipped at any depth.
	// Matching is exact and case-sensitive. ".git" is always skipped, even
	// if it is not listed here.
	ExcludeDirs []string
}

// DefaultOptions returns the built-in settings: a 10 MiB threshold and the
// usual dependency and build-output directories excluded.
func DefaultOptions() Options {
	return Options{
		MaxFileSize: 10 << 20, // 10 MiB
		ExcludeDirs: []string{"node_modules", "vendor", "target", "dist", "build"},
	}
}

// Run scans the directory at root, which must be an absolute path
// (the cli package guarantees this).
//
// Problems found in the repository are reported as Checks. The returned
// error is only for cases where the scan itself cannot happen.
func Run(root string, opts Options) (Result, error) {
	// A zero Options{} would report every non-empty file, which is never
	// what a caller means. Fail loudly instead.
	if opts.MaxFileSize <= 0 {
		return Result{}, errors.New("scan: MaxFileSize must be positive")
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return Result{}, fmt.Errorf("cannot read repository directory: %w", err)
	}

	large, unreadable, err := findLargeFiles(root, opts)
	if err != nil {
		return Result{}, fmt.Errorf("scanning files: %w", err)
	}

	gitRoot := findGitRoot(root)
	types := detectProjectTypes(entries)

	// A plain list keeps the order fixed and the code easy to follow.
	checks := []Check{
		checkGit(gitRoot),
		checkReadme(entries),
		checkLicense(entries),
		checkGitignore(entries),
		checkCI(root),
		checkProjectType(types),
		checkLargeFiles(large, unreadable, opts.MaxFileSize),
	}

	return Result{
		Path:         root,
		GitRoot:      gitRoot,
		ProjectTypes: types,
		Checks:       checks,
		LargeFiles:   large,
		Unreadable:   unreadable,
		Summary:      summarize(checks),
	}, nil
}

func summarize(checks []Check) Summary {
	var s Summary
	for _, c := range checks {
		switch c.Status {
		case StatusPass:
			s.Pass++
		case StatusWarn:
			s.Warn++
		case StatusInfo:
			s.Info++
		}
	}
	return s
}

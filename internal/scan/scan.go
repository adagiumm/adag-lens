// Package scan inspects a repository directory and produces a Result.
// It performs no printing: rendering is the report package's job.
package scan

import (
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

// Result is everything a scan learned about a repository.
type Result struct {
	Path         string   // directory that was scanned
	GitRoot      string   // top of the Git working tree; empty if none found
	ProjectTypes []string // detected project types, in a fixed order
	Checks       []Check  // always in the same order
	Summary      Summary
}

// Run scans the directory at root, which must be an absolute path
// (the cli package guarantees this).
//
// Problems found in the repository are reported as Checks. The returned
// error is only for cases where the scan itself cannot happen.
func Run(root string) (Result, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return Result{}, fmt.Errorf("cannot read repository directory: %w", err)
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
	}

	return Result{
		Path:         root,
		GitRoot:      gitRoot,
		ProjectTypes: types,
		Checks:       checks,
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

package scan

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// findFile returns the name of the first non-directory entry that matches
// one of the candidates, ignoring case. Candidates are tried in order, so
// the result does not depend on directory ordering.
func findFile(entries []os.DirEntry, candidates ...string) (string, bool) {
	for _, c := range candidates {
		for _, e := range entries {
			if !e.IsDir() && strings.EqualFold(e.Name(), c) {
				return e.Name(), true
			}
		}
	}
	return "", false
}

// findGitRoot walks up from start looking for a ".git" entry. It matches a
// directory (normal repository) or a file (worktrees and submodules).
// Any Lstat error counts as "not here", so an unreadable parent
// directory cannot make the scan fail.
func findGitRoot(start string) string {
	dir := start
	for {
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir { // reached the filesystem root
			return ""
		}
		dir = parent
	}
}

func checkGit(gitRoot string) Check {
	if gitRoot == "" {
		return Check{ID: "git", Status: StatusInfo, Message: "Not inside a Git working tree"}
	}
	return Check{ID: "git", Status: StatusPass, Message: "Git working tree detected"}
}

func checkReadme(entries []os.DirEntry) Check {
	name, ok := findFile(entries,
		"README.md", "README", "README.txt", "README.rst", "README.markdown", "README.adoc")
	if !ok {
		return Check{ID: "readme", Status: StatusWarn, Message: "No README found"}
	}
	return Check{ID: "readme", Status: StatusPass, Message: fmt.Sprintf("README found (%s)", name)}
}

func checkLicense(entries []os.DirEntry) Check {
	name, ok := findFile(entries,
		"LICENSE", "LICENSE.md", "LICENSE.txt", "COPYING", "COPYING.md", "UNLICENSE")
	if !ok {
		return Check{ID: "license", Status: StatusWarn, Message: "No license file detected"}
	}
	return Check{ID: "license", Status: StatusPass, Message: fmt.Sprintf("License file found (%s)", name)}
}

func checkGitignore(entries []os.DirEntry) Check {
	if _, ok := findFile(entries, ".gitignore"); !ok {
		return Check{ID: "gitignore", Status: StatusWarn, Message: "No .gitignore found"}
	}
	return Check{ID: "gitignore", Status: StatusPass, Message: ".gitignore found"}
}

// checkCI looks for at least one .yml/.yaml file in .github/workflows.
// Other CI systems are not detected yet, so a missing result is INFO, not
// WARN: the repository may simply use a different CI service.
func checkCI(root string) Check {
	dir := filepath.Join(root, ".github", "workflows")
	files, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return Check{ID: "ci", Status: StatusInfo, Message: "No GitHub Actions workflows found"}
	}
	if err != nil {
		return Check{
			ID:      "ci",
			Status:  StatusInfo,
			Message: "Could not read .github/workflows",
			Details: []string{err.Error()},
		}
	}

	count := 0
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f.Name()))
		if !f.IsDir() && (ext == ".yml" || ext == ".yaml") {
			count++
		}
	}
	if count == 0 {
		return Check{ID: "ci", Status: StatusInfo, Message: "No GitHub Actions workflows found"}
	}
	return Check{ID: "ci", Status: StatusPass, Message: "GitHub Actions configuration found"}
}

// detectProjectTypes maps well-known marker files to project types. It uses
// a slice instead of a map because map iteration order is random in Go and
// we want the same output on every run.
func detectProjectTypes(entries []os.DirEntry) []string {
	markers := []struct {
		file string
		kind string
	}{
		{"go.mod", "Go"},
		{"Cargo.toml", "Rust"},
		{"package.json", "Node.js"},
		{"pyproject.toml", "Python"},
		{"CMakeLists.txt", "C/C++ (CMake)"},
	}

	var types []string
	for _, m := range markers {
		if _, ok := findFile(entries, m.file); ok {
			types = append(types, m.kind)
		}
	}
	return types
}

func checkProjectType(types []string) Check {
	if len(types) == 0 {
		return Check{ID: "project-type", Status: StatusInfo, Message: "Project type not detected"}
	}
	return Check{
		ID:      "project-type",
		Status:  StatusInfo,
		Message: "Project type: " + strings.Join(types, ", "),
	}
}

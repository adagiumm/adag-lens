package scan

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// makeTree creates a temporary directory containing the given paths.
// A path ending in "/" creates an empty directory; anything else creates
// a small file (with parent directories as needed).
func makeTree(t *testing.T, paths ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, p := range paths {
		full := filepath.Join(root, filepath.FromSlash(p))
		if strings.HasSuffix(p, "/") {
			if err := os.MkdirAll(full, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// checkByID returns the check with the given ID, failing the test if absent.
func checkByID(t *testing.T, r Result, id string) Check {
	t.Helper()
	for _, c := range r.Checks {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("no check with ID %q in result", id)
	return Check{}
}

func TestRunHealthyRepo(t *testing.T) {
	root := makeTree(t,
		".git/", "README.md", "LICENSE", ".gitignore",
		".github/workflows/ci.yml", "go.mod")

	r, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}

	want := Summary{Pass: 5, Warn: 0, Info: 1} // project type is always INFO
	if r.Summary != want {
		t.Errorf("summary = %+v, want %+v", r.Summary, want)
	}
	if r.GitRoot != root {
		t.Errorf("GitRoot = %q, want %q", r.GitRoot, root)
	}
	if !reflect.DeepEqual(r.ProjectTypes, []string{"Go"}) {
		t.Errorf("ProjectTypes = %v, want [Go]", r.ProjectTypes)
	}
}

// An empty directory must not crash and must produce the three warnings.
// The "git" check is deliberately not asserted: TMPDIR could, on some
// machines, live inside a Git repository, which would make it flaky.
func TestRunEmptyDirectory(t *testing.T) {
	r, err := Run(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"readme", "license", "gitignore"} {
		if got := checkByID(t, r, id).Status; got != StatusWarn {
			t.Errorf("%s status = %q, want %q", id, got, StatusWarn)
		}
	}
	if r.Summary.Warn != 3 {
		t.Errorf("warnings = %d, want 3", r.Summary.Warn)
	}
}

func TestRunMissingDirectory(t *testing.T) {
	_, err := Run(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected an error for a missing directory")
	}
}

func TestFileChecks(t *testing.T) {
	tests := []struct {
		name  string
		id    string
		files []string
		want  Status
	}{
		{"README.md", "readme", []string{"README.md"}, StatusPass},
		{"lowercase readme", "readme", []string{"readme.md"}, StatusPass},
		{"plain README", "readme", []string{"README"}, StatusPass},
		{"README.rst", "readme", []string{"README.rst"}, StatusPass},
		{"unrelated name", "readme", []string{"README-old.bak"}, StatusWarn},
		{"directory named README", "readme", []string{"README/"}, StatusWarn},
		{"LICENSE", "license", []string{"LICENSE"}, StatusPass},
		{"LICENSE.md", "license", []string{"LICENSE.md"}, StatusPass},
		{"COPYING", "license", []string{"COPYING"}, StatusPass},
		{"no license", "license", []string{"main.go"}, StatusWarn},
		{".gitignore present", "gitignore", []string{".gitignore"}, StatusPass},
		{".gitignore missing", "gitignore", []string{"main.go"}, StatusWarn},
		{"workflow yml", "ci", []string{".github/workflows/ci.yml"}, StatusPass},
		{"workflow uppercase YAML", "ci", []string{".github/workflows/build.YAML"}, StatusPass},
		{"empty workflows dir", "ci", []string{".github/workflows/"}, StatusInfo},
		{"non-yaml in workflows", "ci", []string{".github/workflows/notes.txt"}, StatusInfo},
		{"no .github at all", "ci", []string{"main.go"}, StatusInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := Run(makeTree(t, tt.files...))
			if err != nil {
				t.Fatal(err)
			}
			if got := checkByID(t, r, tt.id).Status; got != tt.want {
				t.Errorf("%s status = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}

func TestProjectTypes(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  []string
	}{
		{"none", []string{"notes.txt"}, nil},
		{"go", []string{"go.mod"}, []string{"Go"}},
		{"python", []string{"pyproject.toml"}, []string{"Python"}},
		{"cmake", []string{"CMakeLists.txt"}, []string{"C/C++ (CMake)"}},
		// Order follows the marker list, not the order files were created.
		{"mixed", []string{"package.json", "go.mod"}, []string{"Go", "Node.js"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := Run(makeTree(t, tt.files...))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(r.ProjectTypes, tt.want) {
				t.Errorf("ProjectTypes = %v, want %v", r.ProjectTypes, tt.want)
			}
		})
	}
}

func TestFindGitRoot(t *testing.T) {
	t.Run("from a nested subdirectory", func(t *testing.T) {
		root := makeTree(t, ".git/", "sub/deeper/")
		if got := findGitRoot(filepath.Join(root, "sub", "deeper")); got != root {
			t.Errorf("findGitRoot = %q, want %q", got, root)
		}
	})

	t.Run(".git as a file (worktree style)", func(t *testing.T) {
		root := makeTree(t, ".git") // a file, not a directory
		if got := findGitRoot(root); got != root {
			t.Errorf("findGitRoot = %q, want %q", got, root)
		}
	})
}

func TestRunIsDeterministic(t *testing.T) {
	root := makeTree(t, ".git/", "README.md", "go.mod", "package.json")
	first, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		again, err := Run(root)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(first, again) {
			t.Fatalf("run %d differs from the first run", i+2)
		}
	}
}

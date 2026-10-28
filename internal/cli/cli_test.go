package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adagiumm/adag-lens/internal/scan"
)

func TestRun(t *testing.T) {
	dir := t.TempDir()

	// A repository with no warnings. The space in the name is deliberate.
	healthy := filepath.Join(dir, "my project")
	for _, d := range []string{".git", filepath.Join(".github", "workflows")} {
		if err := os.MkdirAll(filepath.Join(healthy, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{"README.md", "LICENSE", ".gitignore", filepath.Join(".github", "workflows", "ci.yml")} {
		if err := os.WriteFile(filepath.Join(healthy, f), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// A directory that triggers warnings (no README, license or .gitignore).
	bare := filepath.Join(dir, "bare")
	if err := os.Mkdir(bare, 0o755); err != nil {
		t.Fatal(err)
	}

	// Two 2 KiB files, one of them inside a directory called "cache".
	sized := filepath.Join(dir, "sized")
	if err := os.MkdirAll(filepath.Join(sized, "cache"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"blob.bin", filepath.Join("cache", "cached.bin")} {
		if err := os.WriteFile(filepath.Join(sized, f), make([]byte, 2048), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string // substring; empty means "don't check"
		wantStderr string // substring; empty means "stderr must be empty"
	}{
		{"help", []string{"--help"}, 0, "Usage:", ""},
		{"healthy repository", []string{healthy}, 0, "my project", ""},
		{"explicit text format", []string{"--format", "text", healthy}, 0, "Summary:", ""},
		{"warnings give exit 1", []string{bare}, 1, "No README found", ""},

		{"default threshold", []string{sized}, 1, "No files exceed 10.0 MiB", ""},
		{"custom threshold", []string{"--max-file-size", "1KiB", sized}, 1, "2 files exceed 1.0 KiB", ""},
		{"threshold with equals sign", []string{"--max-file-size=1KiB", sized}, 1, "2 files exceed 1.0 KiB", ""},
		{"exclude one directory", []string{"--max-file-size", "1KiB", "--exclude-dir", "cache", sized}, 1, "1 file exceeds 1.0 KiB", ""},
		{"repeated exclude flag", []string{"--max-file-size", "1KiB", "--exclude-dir", "other", "--exclude-dir", "cache", sized}, 1, "1 file exceeds 1.0 KiB", ""},

		{"missing path", []string{filepath.Join(dir, "nope")}, 2, "", "does not exist"},
		{"path is a file", []string{file}, 2, "", "not a directory"},
		{"bad format", []string{"--format", "xml", healthy}, 2, "", "unsupported format"},
		{"unknown flag", []string{"--bogus"}, 2, "", "bogus"},
		{"too many paths", []string{healthy, healthy}, 2, "", "at most one path"},
		{"decimal unit rejected", []string{"--max-file-size", "10MB", healthy}, 2, "", "unknown unit"},
		{"zero size rejected", []string{"--max-file-size", "0", healthy}, 2, "", "greater than zero"},
		{"size flag without value", []string{"--max-file-size"}, 2, "", "max-file-size"},
		{"exclude path rejected", []string{"--exclude-dir", "src/generated", healthy}, 2, "", "looks like a path"},
		{"exclude empty rejected", []string{"--exclude-dir", "", healthy}, 2, "", "must not be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(tt.args, &stdout, &stderr)

			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d (stderr: %q)", code, tt.wantCode, stderr.String())
			}
			if tt.wantStdout != "" && !strings.Contains(stdout.String(), tt.wantStdout) {
				t.Errorf("stdout = %q, want it to contain %q", stdout.String(), tt.wantStdout)
			}
			if tt.wantStderr == "" && stderr.Len() > 0 {
				t.Errorf("unexpected stderr: %q", stderr.String())
			}
			if tt.wantStderr != "" && !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("stderr = %q, want it to contain %q", stderr.String(), tt.wantStderr)
			}
		})
	}
}

// The default exclusions appear both in scan.DefaultOptions and in the help
// text. This keeps the two from drifting apart.
func TestUsageMentionsDefaultExclusions(t *testing.T) {
	for _, name := range scan.DefaultOptions().ExcludeDirs {
		if !strings.Contains(usageText, name) {
			t.Errorf("usage text does not mention default excluded directory %q", name)
		}
	}
}

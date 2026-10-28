package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	dir := t.TempDir()

	spaced := filepath.Join(dir, "my project")
	if err := os.Mkdir(spaced, 0o755); err != nil {
		t.Fatal(err)
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
		{"valid directory", []string{dir}, 0, dir, ""},
		{"path with spaces", []string{spaced}, 0, "my project", ""},
		{"explicit text format", []string{"--format", "text", dir}, 0, dir, ""},
		{"missing path", []string{filepath.Join(dir, "nope")}, 2, "", "does not exist"},
		{"path is a file", []string{file}, 2, "", "not a directory"},
		{"bad format", []string{"--format", "xml", dir}, 2, "", "unsupported format"},
		{"unknown flag", []string{"--bogus"}, 2, "", "bogus"},
		{"too many paths", []string{dir, dir}, 2, "", "at most one path"},
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

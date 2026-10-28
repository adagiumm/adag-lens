package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeRepo builds a small repository with README, license and .gitignore,
// plus a big.bin file of bigSize bytes when bigSize > 0.
func makeRepo(t *testing.T, bigSize int) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"README.md", "LICENSE", ".gitignore"} {
		if err := os.WriteFile(filepath.Join(root, f), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if bigSize > 0 {
		if err := os.WriteFile(filepath.Join(root, "big.bin"), make([]byte, bigSize), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestRunJSONWithWarning(t *testing.T) {
	repo := makeRepo(t, 2048)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--format", "json", "--max-file-size", "1KiB", repo}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("exit code = %d, want 1 (stderr: %q)", code, stderr.String())
	}
	if stderr.Len() > 0 {
		t.Errorf("unexpected stderr: %q", stderr.String())
	}
	out := stdout.String()
	if strings.Contains(out, "\x1b") || strings.Contains(out, "scanned in") {
		t.Errorf("JSON output must not contain colors or the text report's duration:\n%s", out)
	}

	var doc struct {
		SchemaVersion int `json:"schema_version"`
		Repository    struct {
			Path string `json:"path"`
		} `json:"repository"`
		LargeFiles []struct {
			Path      string `json:"path"`
			SizeBytes int64  `json:"size_bytes"`
		} `json:"large_files"`
		Summary struct {
			Pass int `json:"pass"`
			Warn int `json:"warn"`
			Info int `json:"info"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, out)
	}

	if doc.SchemaVersion != 1 {
		t.Errorf("schema_version = %d, want 1", doc.SchemaVersion)
	}
	if doc.Repository.Path != repo {
		t.Errorf("repository.path = %q, want %q", doc.Repository.Path, repo)
	}
	if len(doc.LargeFiles) != 1 || doc.LargeFiles[0].Path != "big.bin" || doc.LargeFiles[0].SizeBytes != 2048 {
		t.Errorf("large_files = %+v, want one entry: big.bin, 2048 bytes", doc.LargeFiles)
	}
	// git, readme, license, gitignore pass; ci and project type are info; large-files warns.
	if doc.Summary.Pass != 4 || doc.Summary.Warn != 1 || doc.Summary.Info != 2 {
		t.Errorf("summary = %+v, want pass=4 warn=1 info=2", doc.Summary)
	}
}

func TestRunJSONWithoutWarnings(t *testing.T) {
	repo := makeRepo(t, 0)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--format", "json", repo}, &stdout, &stderr)

	if code != 0 {
		t.Errorf("exit code = %d, want 0 (stderr: %q)", code, stderr.String())
	}
	if !json.Valid(stdout.Bytes()) {
		t.Errorf("stdout is not valid JSON:\n%s", stdout.String())
	}
}

// On a fatal error stdout must stay empty, so `adag-lens --format json . > out.json`
// never leaves a misleading file behind.
func TestRunJSONFatalErrorKeepsStdoutEmpty(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--format", "json", filepath.Join(t.TempDir(), "nope")}, &stdout, &stderr)

	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout must be empty on a fatal error, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "does not exist") {
		t.Errorf("stderr = %q, want it to explain the problem", stderr.String())
	}
}

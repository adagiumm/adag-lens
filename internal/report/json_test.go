package report

import (
	"encoding/json"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/adagiumm/adag-lens/internal/scan"
)

// renderJSON runs WriteJSON, checks that the output is valid JSON, and
// returns both the raw text and the decoded generic document. Decoding into
// map[string]any shows what a script would see, including [] versus null.
func renderJSON(t *testing.T, r scan.Result) (string, map[string]any) {
	t.Helper()
	var b strings.Builder
	if err := WriteJSON(&b, r); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if !json.Valid([]byte(out)) {
		t.Fatalf("output is not valid JSON:\n%s", out)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	return out, doc
}

func TestWriteJSONFullDocument(t *testing.T) {
	r := sampleResult()
	r.ProjectTypes = []string{"Go"}
	r.Checks[1].Details = []string{"detail one"}
	r.LargeFiles = []scan.LargeFile{{Path: "big.bin", Size: 11534336}}
	r.Unreadable = []string{"locked"}

	_, doc := renderJSON(t, r)

	// JSON numbers decode to float64, hence the 1.0 literals.
	want := map[string]any{
		"schema_version": 1.0,
		"repository": map[string]any{
			"path":          "/repo",
			"git_root":      "/repo",
			"project_types": []any{"Go"},
		},
		"checks": []any{
			map[string]any{"id": "git", "status": "pass", "message": "Git working tree detected", "details": []any{}},
			map[string]any{"id": "readme", "status": "warn", "message": "No README found", "details": []any{"detail one"}},
			map[string]any{"id": "ci", "status": "info", "message": "No GitHub Actions workflows found", "details": []any{}},
		},
		"large_files":      []any{map[string]any{"path": "big.bin", "size_bytes": 11534336.0}},
		"unreadable_paths": []any{"locked"},
		"summary":          map[string]any{"pass": 1.0, "warn": 1.0, "info": 1.0},
	}
	if !reflect.DeepEqual(doc, want) {
		t.Errorf("document mismatch\n got: %v\nwant: %v", doc, want)
	}
}

// Every list must be [] and never null, and git_root must be present as null.
func TestWriteJSONEmptyResult(t *testing.T) {
	_, doc := renderJSON(t, scan.Result{Path: "/repo"})

	want := map[string]any{
		"schema_version": 1.0,
		"repository": map[string]any{
			"path":          "/repo",
			"git_root":      nil,
			"project_types": []any{},
		},
		"checks":           []any{},
		"large_files":      []any{},
		"unreadable_paths": []any{},
		"summary":          map[string]any{"pass": 0.0, "warn": 0.0, "info": 0.0},
	}
	if !reflect.DeepEqual(doc, want) {
		t.Errorf("document mismatch\n got: %v\nwant: %v", doc, want)
	}
}

// The set of top-level keys is part of the contract: adding or removing one
// by accident should fail a test.
func TestWriteJSONTopLevelKeys(t *testing.T) {
	_, doc := renderJSON(t, sampleResult())
	want := []string{"checks", "large_files", "repository", "schema_version", "summary", "unreadable_paths"}
	if got := slices.Sorted(maps.Keys(doc)); !slices.Equal(got, want) {
		t.Errorf("top-level keys = %v, want %v", got, want)
	}
}

func TestWriteJSONEscaping(t *testing.T) {
	r := sampleResult()
	r.Checks[1].Message = "bad\x1b[2J<b> & name"

	out, _ := renderJSON(t, r)

	if strings.Contains(out, "\x1b") {
		t.Errorf("raw escape character leaked into JSON output: %q", out)
	}
	if !strings.Contains(out, `\u001b`) {
		t.Errorf("escape character should be encoded as \\u001b:\n%s", out)
	}
	if !strings.Contains(out, "<b> & name") {
		t.Errorf("HTML characters should not be escaped:\n%s", out)
	}

	// The message must survive a round trip unchanged.
	var got struct {
		Checks []struct {
			Message string `json:"message"`
		} `json:"checks"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if want := "bad\x1b[2J<b> & name"; got.Checks[1].Message != want {
		t.Errorf("message = %q, want %q", got.Checks[1].Message, want)
	}
}

func TestWriteJSONFormatting(t *testing.T) {
	out, _ := renderJSON(t, sampleResult())
	if !strings.HasPrefix(out, "{\n  \"schema_version\": 1,\n") {
		t.Errorf("expected 2-space indented output starting with schema_version, got:\n%s", out)
	}
	if !strings.HasSuffix(out, "}\n") {
		t.Errorf("output should end with a closing brace and a newline, got:\n%q", out)
	}
}

func TestWriteJSONReturnsWriteError(t *testing.T) {
	if err := WriteJSON(failingWriter{}, sampleResult()); err == nil {
		t.Error("expected the writer's error to be returned")
	}
}

package report

import (
	"encoding/json"
	"io"

	"github.com/adagiumm/adag-lens/internal/scan"
)

// SchemaVersion identifies the JSON layout. Adding fields keeps the
// version; renaming or removing a field, or changing its meaning, bumps it.
// See docs/json-output.md.
const SchemaVersion = 1

// The types below are the public JSON contract. They are separate from
// scan.Result on purpose: renaming an internal field must never silently
// change the output that other programs depend on.

type jsonReport struct {
	SchemaVersion   int             `json:"schema_version"`
	Repository      jsonRepository  `json:"repository"`
	Checks          []jsonCheck     `json:"checks"`
	LargeFiles      []jsonLargeFile `json:"large_files"`
	UnreadablePaths []string        `json:"unreadable_paths"`
	Summary         jsonSummary     `json:"summary"`
}

type jsonRepository struct {
	Path         string   `json:"path"`
	GitRoot      *string  `json:"git_root"` // nil encodes as null: "no Git working tree"
	ProjectTypes []string `json:"project_types"`
}

type jsonCheck struct {
	ID      string   `json:"id"`
	Status  string   `json:"status"`
	Message string   `json:"message"`
	Details []string `json:"details"`
}

type jsonLargeFile struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
}

type jsonSummary struct {
	Pass int `json:"pass"`
	Warn int `json:"warn"`
	Info int `json:"info"`
}

// WriteJSON writes the scan result to w as indented JSON followed by a
// newline. It contains no colors, no durations and nothing else that varies
// between runs on the same repository.
func WriteJSON(w io.Writer, r scan.Result) error {
	out := jsonReport{
		SchemaVersion: SchemaVersion,
		Repository: jsonRepository{
			Path:         r.Path,
			ProjectTypes: orEmpty(r.ProjectTypes),
		},
		Checks:          make([]jsonCheck, 0, len(r.Checks)),
		LargeFiles:      make([]jsonLargeFile, 0, len(r.LargeFiles)),
		UnreadablePaths: orEmpty(r.Unreadable),
		Summary:         jsonSummary{Pass: r.Summary.Pass, Warn: r.Summary.Warn, Info: r.Summary.Info},
	}
	if r.GitRoot != "" {
		out.Repository.GitRoot = &r.GitRoot
	}
	for _, c := range r.Checks {
		out.Checks = append(out.Checks, jsonCheck{
			ID:      c.ID,
			Status:  string(c.Status),
			Message: c.Message,
			Details: orEmpty(c.Details),
		})
	}
	for _, f := range r.LargeFiles {
		out.LargeFiles = append(out.LargeFiles, jsonLargeFile{Path: f.Path, SizeBytes: f.Size})
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false) // keep <, > and & readable in file names and messages
	return enc.Encode(out)
}

// orEmpty turns a nil slice into an empty one. Go encodes a nil slice as
// JSON null, which would break scripts that expect an array.
func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

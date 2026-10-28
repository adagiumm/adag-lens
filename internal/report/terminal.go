// Package report renders a scan.Result for people (and, later, for
// programs). It never touches the filesystem.
package report

import (
	"fmt"
	"io"
	"strings"
	"time"
	"unicode"

	"github.com/adagiumm/adag-lens/internal/scan"
)

// TextOptions controls how WriteText renders the report.
type TextOptions struct {
	// Color adds ANSI colors to the status labels. The report is complete
	// without them: the label text itself carries the meaning.
	Color bool

	// Duration is how long the scan took. Zero means "don't print it",
	// which keeps tests and golden outputs deterministic.
	Duration time.Duration
}

// WriteText writes the human-readable report to w.
func WriteText(w io.Writer, r scan.Result, opts TextOptions) error {
	var b strings.Builder

	b.WriteString("adag-lens - Repository Health\n\n")
	fmt.Fprintf(&b, "Repository: %s\n", sanitize(r.Path))
	// Only worth showing when the scanned path is inside a larger repository.
	if r.GitRoot != "" && r.GitRoot != r.Path {
		fmt.Fprintf(&b, "Git root:   %s\n", sanitize(r.GitRoot))
		b.WriteString("Note:       scanning a subdirectory; run on the Git root for repository-level checks\n")
	}
	b.WriteString("\n")

	for _, c := range r.Checks {
		fmt.Fprintf(&b, "%s %s\n", label(c.Status, opts.Color), sanitize(c.Message))
		for _, d := range c.Details {
			fmt.Fprintf(&b, "       %s\n", sanitize(d)) // aligned under the message
		}
	}

	s := r.Summary
	fmt.Fprintf(&b, "\nSummary: %d passed, %s, %d info", s.Pass, plural(s.Warn, "warning"), s.Info)
	if opts.Duration > 0 {
		fmt.Fprintf(&b, " (scanned in %s)", formatDuration(opts.Duration))
	}
	b.WriteString("\n")

	_, err := io.WriteString(w, b.String())
	return err
}

// label returns "[PASS]", "[WARN]" or "[INFO]", optionally colored.
// All three have the same width, so messages line up.
func label(s scan.Status, color bool) string {
	text := "[" + strings.ToUpper(string(s)) + "]"
	if !color {
		return text
	}

	var code string
	switch s {
	case scan.StatusPass:
		code = "32" // green
	case scan.StatusWarn:
		code = "33" // yellow
	case scan.StatusInfo:
		code = "36" // cyan
	default:
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

// sanitize replaces control characters with '?'. File names come from the
// repository being inspected, so they are untrusted: a name containing a
// terminal escape sequence could otherwise rewrite what the user sees.
func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return '?'
		}
		return r
	}, s)
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return "<1ms"
	}
	return d.Round(time.Millisecond).String()
}

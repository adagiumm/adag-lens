package report

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/adagiumm/adag-lens/internal/scan"
)

// sampleResult is built by hand: the renderer needs no filesystem.
func sampleResult() scan.Result {
	return scan.Result{
		Path:    "/repo",
		GitRoot: "/repo",
		Checks: []scan.Check{
			{ID: "git", Status: scan.StatusPass, Message: "Git working tree detected"},
			{ID: "readme", Status: scan.StatusWarn, Message: "No README found"},
			{ID: "ci", Status: scan.StatusInfo, Message: "No GitHub Actions workflows found"},
		},
		Summary: scan.Summary{Pass: 1, Warn: 1, Info: 1},
	}
}

func render(t *testing.T, r scan.Result, opts TextOptions) string {
	t.Helper()
	var b strings.Builder
	if err := WriteText(&b, r, opts); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func TestWriteTextPlain(t *testing.T) {
	want := "adag-lens - Repository Health\n" +
		"\n" +
		"Repository: /repo\n" +
		"\n" +
		"[PASS] Git working tree detected\n" +
		"[WARN] No README found\n" +
		"[INFO] No GitHub Actions workflows found\n" +
		"\n" +
		"Summary: 1 passed, 1 warning, 1 info\n"

	if got := render(t, sampleResult(), TextOptions{}); got != want {
		t.Errorf("output mismatch\n got:\n%s\nwant:\n%s", got, want)
	}
}

func TestWriteTextColor(t *testing.T) {
	colored := render(t, sampleResult(), TextOptions{Color: true})
	if !strings.Contains(colored, "\x1b[33m[WARN]\x1b[0m") {
		t.Errorf("expected a colored WARN label, got:\n%q", colored)
	}

	plain := render(t, sampleResult(), TextOptions{Color: false})
	if strings.Contains(plain, "\x1b") {
		t.Errorf("plain output must not contain escape codes, got:\n%q", plain)
	}
}

func TestWriteTextGitRootAndDetails(t *testing.T) {
	r := sampleResult()
	r.Path = "/repo/sub"
	r.Checks[1].Details = []string{"first line", "second line"}

	got := render(t, r, TextOptions{})
	for _, want := range []string{
		"Git root:   /repo\n",
		"scanning a subdirectory",
		"       first line\n",
		"       second line\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestWriteTextStripsControlCharacters(t *testing.T) {
	r := sampleResult()
	r.Checks[1].Message = "bad\x1b[2Jname\nsecond line"

	got := render(t, r, TextOptions{})
	if strings.Contains(got, "\x1b") {
		t.Errorf("escape character leaked into output: %q", got)
	}
	if !strings.Contains(got, "bad?[2Jname?second line") {
		t.Errorf("expected control characters replaced by '?', got:\n%s", got)
	}
}

func TestWriteTextSummaryWording(t *testing.T) {
	tests := []struct {
		warn int
		want string
	}{
		{0, "0 warnings"},
		{1, "1 warning,"},
		{2, "2 warnings"},
	}
	for _, tt := range tests {
		r := sampleResult()
		r.Summary.Warn = tt.warn
		if got := render(t, r, TextOptions{}); !strings.Contains(got, tt.want) {
			t.Errorf("Warn=%d: output missing %q:\n%s", tt.warn, tt.want, got)
		}
	}
}

func TestWriteTextDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{250 * time.Millisecond, "(scanned in 250ms)"},
		{500 * time.Microsecond, "(scanned in <1ms)"},
	}
	for _, tt := range tests {
		if got := render(t, sampleResult(), TextOptions{Duration: tt.d}); !strings.Contains(got, tt.want) {
			t.Errorf("Duration=%v: output missing %q:\n%s", tt.d, tt.want, got)
		}
	}

	if got := render(t, sampleResult(), TextOptions{}); strings.Contains(got, "scanned in") {
		t.Errorf("zero duration must not be printed:\n%s", got)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestWriteTextReturnsWriteError(t *testing.T) {
	if err := WriteText(failingWriter{}, sampleResult(), TextOptions{}); err == nil {
		t.Error("expected the writer's error to be returned")
	}
}

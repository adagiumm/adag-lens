package report

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// File names on Linux can contain bytes that are not valid UTF-8. They must
// never reach the output unchanged: terminals may misinterpret them, and
// JSON can only carry valid UTF-8.

func TestWriteTextReplacesInvalidUTF8(t *testing.T) {
	r := sampleResult()
	r.Checks[1].Message = "bad\xffname"

	got := render(t, r, TextOptions{})

	if strings.Contains(got, "\xff") || !utf8.ValidString(got) {
		t.Errorf("invalid UTF-8 leaked into the text report: %q", got)
	}
	if !strings.Contains(got, "bad\uFFFDname") {
		t.Errorf("expected the replacement character, got:\n%s", got)
	}
}

func TestWriteJSONReplacesInvalidUTF8(t *testing.T) {
	r := sampleResult()
	r.Checks[1].Message = "bad\xffname"

	out, _ := renderJSON(t, r)

	if !utf8.ValidString(out) {
		t.Errorf("JSON output is not valid UTF-8: %q", out)
	}
	if !strings.Contains(out, `bad\ufffdname`) {
		t.Errorf("expected \\ufffd in the JSON output, got:\n%s", out)
	}
}

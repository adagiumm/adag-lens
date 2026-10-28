package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--version"}, &stdout, &stderr)

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.HasPrefix(stdout.String(), "adag-lens ") {
		t.Errorf("stdout = %q, want it to start with \"adag-lens \"", stdout.String())
	}
	if stderr.Len() > 0 {
		t.Errorf("unexpected stderr: %q", stderr.String())
	}
}

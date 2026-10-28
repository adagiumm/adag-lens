package version

import "testing"

func TestStringUsesBuildTimeVersion(t *testing.T) {
	old := Version
	t.Cleanup(func() { Version = old })

	Version = "v1.2.3"
	if got := String(); got != "v1.2.3" {
		t.Errorf("String() = %q, want v1.2.3", got)
	}
}

func TestStringHasFallback(t *testing.T) {
	old := Version
	t.Cleanup(func() { Version = old })

	Version = ""
	if got := String(); got == "" {
		t.Error("String() must never be empty")
	}
}

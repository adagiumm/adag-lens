package scan

import (
	"strings"
	"testing"
)

func TestParseSize(t *testing.T) {
	valid := []struct {
		in   string
		want int64
	}{
		{"10MiB", 10 << 20},
		{"10mib", 10 << 20},
		{"500KiB", 500 << 10},
		{"1GiB", 1 << 30},
		{"1024", 1024},
		{"1024B", 1024},
		{"1b", 1},
		{" 5 MiB ", 5 << 20},
	}
	for _, tt := range valid {
		t.Run("valid "+tt.in, func(t *testing.T) {
			got, err := ParseSize(tt.in)
			if err != nil {
				t.Fatalf("ParseSize(%q) returned error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("ParseSize(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}

	invalid := []struct {
		in      string
		wantMsg string // substring expected in the error
	}{
		{"", "whole number"},
		{"MiB", "whole number"},
		{"-1MiB", "whole number"},
		{"1.5MiB", "unknown unit"},
		{"10MB", "unknown unit"},
		{"10M", "unknown unit"},
		{"0", "greater than zero"},
		{"0MiB", "greater than zero"},
		{"9999999999999GiB", "too large"},
		{"99999999999999999999", "too large"},
	}
	for _, tt := range invalid {
		t.Run("invalid "+tt.in, func(t *testing.T) {
			_, err := ParseSize(tt.in)
			if err == nil {
				t.Fatalf("ParseSize(%q) expected an error", tt.in)
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("error = %q, want it to contain %q", err, tt.wantMsg)
			}
		})
	}
}

package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

// smallLimit lets tests use tiny files instead of real 10 MiB ones.
func smallLimit() Options {
	o := DefaultOptions()
	o.MaxFileSize = 100
	return o
}

// writeSized creates a file of exactly n bytes at rel (slash-separated)
// under root, creating parent directories as needed.
func writeSized(t *testing.T, root, rel string, n int) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, make([]byte, n), 0o644); err != nil {
		t.Fatal(err)
	}
}

func largePaths(r Result) []string {
	var paths []string
	for _, f := range r.LargeFiles {
		paths = append(paths, f.Path)
	}
	return paths
}

func TestLargeFilesDetected(t *testing.T) {
	root := t.TempDir()
	writeSized(t, root, "small.txt", 100) // exactly at the limit: not reported
	writeSized(t, root, "big.bin", 101)
	writeSized(t, root, "my docs/größer.bin", 500) // spaces and Unicode

	r, err := Run(root, smallLimit())
	if err != nil {
		t.Fatal(err)
	}

	want := []LargeFile{{"my docs/größer.bin", 500}, {"big.bin", 101}} // biggest first
	if !slices.Equal(r.LargeFiles, want) {
		t.Errorf("LargeFiles = %v, want %v", r.LargeFiles, want)
	}

	c := checkByID(t, r, "large-files")
	if c.Status != StatusWarn {
		t.Errorf("status = %q, want %q", c.Status, StatusWarn)
	}
	if c.Message != "2 files exceed 100 B" {
		t.Errorf("message = %q", c.Message)
	}
	if !slices.Contains(c.Details, "big.bin (101 B)") {
		t.Errorf("details missing big.bin: %v", c.Details)
	}
}

func TestNoLargeFiles(t *testing.T) {
	root := t.TempDir()
	writeSized(t, root, "a.txt", 10)

	r, err := Run(root, smallLimit())
	if err != nil {
		t.Fatal(err)
	}
	c := checkByID(t, r, "large-files")
	if c.Status != StatusPass || c.Message != "No files exceed 100 B" {
		t.Errorf("got %q / %q", c.Status, c.Message)
	}
}

func TestExcludedDirectories(t *testing.T) {
	root := t.TempDir()
	writeSized(t, root, ".git/objects/pack.bin", 300)
	writeSized(t, root, "node_modules/lib/big.bin", 300)
	writeSized(t, root, "src/vendor/big.bin", 300) // excluded at any depth
	writeSized(t, root, "build/out.bin", 300)
	writeSized(t, root, "docs/big.bin", 300)
	writeSized(t, root, "dist", 200) // a FILE named like an excluded directory

	r, err := Run(root, smallLimit())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"docs/big.bin", "dist"}
	if got := largePaths(r); !slices.Equal(got, want) {
		t.Errorf("paths = %v, want %v", got, want)
	}
}

func TestGitDirectoryAlwaysExcluded(t *testing.T) {
	root := t.TempDir()
	writeSized(t, root, ".git/huge.bin", 400)
	writeSized(t, root, "ok.bin", 300)

	opts := smallLimit()
	opts.ExcludeDirs = nil

	r, err := Run(root, opts)
	if err != nil {
		t.Fatal(err)
	}
	if got := largePaths(r); !slices.Equal(got, []string{"ok.bin"}) {
		t.Errorf("paths = %v, want [ok.bin]", got)
	}
}

func TestCustomExcludeDirs(t *testing.T) {
	root := t.TempDir()
	writeSized(t, root, "generated/big.bin", 300)
	writeSized(t, root, "node_modules/big.bin", 200)

	opts := smallLimit()
	opts.ExcludeDirs = []string{"generated"} // replaces the defaults

	r, err := Run(root, opts)
	if err != nil {
		t.Fatal(err)
	}
	if got := largePaths(r); !slices.Equal(got, []string{"node_modules/big.bin"}) {
		t.Errorf("paths = %v, want [node_modules/big.bin]", got)
	}
}

func TestRootNamedLikeExcludedDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "build")
	writeSized(t, root, "big.bin", 300)

	r, err := Run(root, smallLimit())
	if err != nil {
		t.Fatal(err)
	}
	if got := largePaths(r); !slices.Equal(got, []string{"big.bin"}) {
		t.Errorf("paths = %v, want [big.bin]", got)
	}
}

func TestSymlinksNotFollowed(t *testing.T) {
	outside := t.TempDir()
	writeSized(t, outside, "huge.bin", 500)
	writeSized(t, outside, "dir/huge2.bin", 500)

	root := t.TempDir()
	if err := os.Symlink(filepath.Join(outside, "huge.bin"), filepath.Join(root, "link-file")); err != nil {
		t.Skipf("symlinks not supported here: %v", err)
	}
	if err := os.Symlink(filepath.Join(outside, "dir"), filepath.Join(root, "link-dir")); err != nil {
		t.Fatal(err)
	}

	r, err := Run(root, smallLimit())
	if err != nil {
		t.Fatal(err)
	}
	if got := largePaths(r); len(got) != 0 {
		t.Errorf("symlinked files must not be reported, got %v", got)
	}
}

func TestUnreadableDirectory(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("permission bits are not enforced here (Windows or root)")
	}

	root := t.TempDir()
	writeSized(t, root, "ok/big.bin", 200)
	writeSized(t, root, "locked/big.bin", 200)

	locked := filepath.Join(root, "locked")
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	// Restore permissions so t.TempDir can clean up. Cleanups run last-in,
	// first-out, so this runs before the temp directory is removed.
	t.Cleanup(func() { os.Chmod(locked, 0o755) })

	r, err := Run(root, smallLimit())
	if err != nil {
		t.Fatalf("an unreadable directory must not fail the scan: %v", err)
	}
	if got := largePaths(r); !slices.Equal(got, []string{"ok/big.bin"}) {
		t.Errorf("paths = %v, want [ok/big.bin]", got)
	}
	if !slices.Equal(r.Unreadable, []string{"locked"}) {
		t.Errorf("Unreadable = %v, want [locked]", r.Unreadable)
	}
	c := checkByID(t, r, "large-files")
	if !slices.Contains(c.Details, "unreadable: locked") {
		t.Errorf("details should name the unreadable path: %v", c.Details)
	}
}

func TestDetailsAreCapped(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 12; i++ {
		writeSized(t, root, fmt.Sprintf("f%02d.bin", i), 200)
	}

	r, err := Run(root, smallLimit())
	if err != nil {
		t.Fatal(err)
	}
	c := checkByID(t, r, "large-files")
	if len(c.Details) != maxDetails+1 {
		t.Fatalf("len(Details) = %d, want %d", len(c.Details), maxDetails+1)
	}
	if last := c.Details[len(c.Details)-1]; last != "... and 2 more" {
		t.Errorf("last detail = %q", last)
	}
	if len(r.LargeFiles) != 12 {
		t.Errorf("Result must keep the full list, got %d files", len(r.LargeFiles))
	}
}

func TestRunRejectsZeroOptions(t *testing.T) {
	if _, err := Run(t.TempDir(), Options{}); err == nil {
		t.Error("expected an error when MaxFileSize is not set")
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1023, "1023 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{10 << 20, "10.0 MiB"},
		{1 << 30, "1.0 GiB"},
	}
	for _, tt := range tests {
		if got := formatSize(tt.in); got != tt.want {
			t.Errorf("formatSize(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

package scan

import (
	"cmp"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
)

// maxDetails caps how many files a check lists, so a repository with
// thousands of large files cannot flood the terminal. The complete list
// stays available in Result.LargeFiles.
const maxDetails = 10

// findLargeFiles walks root and returns files bigger than opts.MaxFileSize
// (biggest first) plus the paths it could not read.
//
// Symlinks are never followed: WalkDir reports them as non-directory
// entries, and the IsRegular check below ignores everything that is not an
// ordinary file (symlinks, sockets, devices). Only metadata is read; file
// contents are never loaded.
func findLargeFiles(root string, opts Options) ([]LargeFile, []string, error) {
	excluded := map[string]bool{".git": true} // always skipped
	for _, name := range opts.ExcludeDirs {
		excluded[name] = true
	}

	var large []LargeFile
	var unreadable []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Permission denied, file removed mid-scan, ...: note it and
			// keep going. Returning nil continues the walk.
			unreadable = append(unreadable, relPath(root, path))
			return nil
		}

		if d.IsDir() {
			// Never skip the root itself: the user may scan a directory
			// that happens to be called "build".
			if path != root && excluded[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}

		if !d.Type().IsRegular() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			unreadable = append(unreadable, relPath(root, path))
			return nil
		}
		if info.Size() > opts.MaxFileSize {
			large = append(large, LargeFile{Path: relPath(root, path), Size: info.Size()})
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	// Biggest first; ties are broken by path so the order never varies.
	slices.SortFunc(large, func(a, b LargeFile) int {
		return cmp.Or(cmp.Compare(b.Size, a.Size), strings.Compare(a.Path, b.Path))
	})
	return large, unreadable, nil
}

// relPath returns path relative to root with forward slashes, so output is
// the same on every operating system.
func relPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func checkLargeFiles(large []LargeFile, unreadable []string, limit int64) Check {
	limitText := formatSize(limit)
	c := Check{ID: "large-files"}

	switch {
	case len(large) == 1:
		c.Status, c.Message = StatusWarn, "1 file exceeds "+limitText
	case len(large) > 1:
		c.Status, c.Message = StatusWarn, fmt.Sprintf("%d files exceed %s", len(large), limitText)
	case len(unreadable) > 0:
		// Saying PASS here would be misleading: we could not look everywhere.
		c.Status = StatusInfo
		c.Message = fmt.Sprintf("No files over %s found, but some paths could not be read", limitText)
	default:
		c.Status, c.Message = StatusPass, "No files exceed "+limitText
	}

	for i, f := range large {
		if i == maxDetails {
			c.Details = append(c.Details, fmt.Sprintf("... and %d more", len(large)-maxDetails))
			break
		}
		c.Details = append(c.Details, fmt.Sprintf("%s (%s)", f.Path, formatSize(f.Size)))
	}
	// Name the paths we could not read (capped, like the large files), so
	// the user knows exactly what was not checked.
	for i, p := range unreadable {
		if i == maxDetails {
			c.Details = append(c.Details, fmt.Sprintf("... and %d more unreadable paths", len(unreadable)-maxDetails))
			break
		}
		c.Details = append(c.Details, "unreadable: "+p)
	}
	return c
}

// formatSize renders a byte count using binary units (KiB, MiB, ...).
func formatSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

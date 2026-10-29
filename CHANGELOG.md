# Changelog

All notable changes to this project are documented here. The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.1.0] - 2026-10-29

First release.

### Added

- `adag-lens [path]` scans a directory (default: the current directory) and prints a report.
- Checks for a Git working tree (including worktrees), README, license file, `.gitignore`, GitHub Actions workflows, and project type (Go, Rust, Node.js, Python, CMake).
- Large-file detection with a configurable threshold (`--max-file-size`, default 10 MiB) and configurable directory exclusions (`--exclude-dir`). Symbolic links are not followed; unreadable paths are reported.
- Human-readable terminal report with `PASS`/`WARN`/`INFO` statuses; optional color that respects `NO_COLOR`.
- JSON output (`--format json`) with a documented schema (`schema_version` 1).
- Exit codes: `0` no warnings, `1` warnings, `2` invalid arguments or fatal error.
- `--version` and `--help`.
- Tests and a GitHub Actions workflow running on Linux, macOS and Windows.
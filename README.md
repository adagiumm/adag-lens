# adag-lens

[![CI](https://github.com/adagiumm/adag-lens/actions/workflows/ci.yml/badge.svg)](https://github.com/adagiumm/adag-lens/actions/workflows/ci.yml)

**Understand the health of any repository in seconds.**

adag-lens is a small, local-first command-line tool that inspects a Git repository and prints a short report about its structure, documentation and basic hygiene: is there a README, a license, a `.gitignore`, CI configuration, and are there unexpectedly large files?

It runs entirely on your machine. It needs no account, makes no network requests, never modifies the repository it inspects, and does not read file contents (it only looks at names and sizes).

> adag-lens checks basic repository hygiene and discoverability. It is **not** a security scanner and does not look for secrets or vulnerabilities.

## What it checks

| Check | Looks for | Status when missing |
|---|---|---|
| Git | a `.git` entry in the scanned directory or any parent | INFO |
| README | `README.md`, `README`, `README.txt`, `README.rst`, `README.markdown`, `README.adoc` | WARN |
| License | `LICENSE`, `LICENSE.md`, `LICENSE.txt`, `COPYING`, `COPYING.md`, `UNLICENSE` | WARN |
| `.gitignore` | a `.gitignore` file | WARN |
| CI | a `.yml` or `.yaml` file in `.github/workflows/` (GitHub Actions only) | INFO |
| Project type | `go.mod`, `Cargo.toml`, `package.json`, `pyproject.toml`, `CMakeLists.txt` | INFO |
| Large files | files bigger than a threshold (default 10 MiB) | WARN |

File names are matched case-insensitively and only in the top-level directory (plus `.github/workflows/` for CI). adag-lens checks that these files **exist**; it does not judge their quality.

Statuses: `PASS` means the check succeeded, `WARN` means something worth fixing, `INFO` is neutral information.

## Installation

You need a recent Go toolchain (the minimum version is the `go` line in [`go.mod`](go.mod)).

Install with Go:

```bash
go install github.com/adagiumm/adag-lens/cmd/adag-lens@latest
```

Or build from source:

```bash
git clone https://github.com/adagiumm/adag-lens.git
cd adag-lens
make build        # or: go build -o adag-lens ./cmd/adag-lens
```

## Usage

```bash
adag-lens [flags] [path]
```

If `path` is omitted, the current directory is used. **Flags must come before the path.**

```bash
adag-lens                               # scan the current directory
adag-lens ~/projects/my-app             # scan another directory
adag-lens --max-file-size 5MiB .        # report files over 5 MiB
adag-lens --exclude-dir generated .     # also skip directories named "generated"
adag-lens --format json . > report.json
```

| Flag | Meaning |
|---|---|
| `--format text\|json` | Output format (default `text`). |
| `--max-file-size SIZE` | Report files larger than this. Units: `B`, `KiB`, `MiB`, `GiB` (binary; `KB` and `MB` are rejected as ambiguous). Default `10MiB`. |
| `--exclude-dir NAME` | Skip directories with this name at any depth. Repeatable. Added to the defaults. |
| `--version` | Print the version and exit. |
| `--help` | Show help. |

### Example output

An illustrative run on a made-up project:

```text
adag-lens - Repository Health

Repository: /home/user/example-project

[PASS] Git working tree detected
[PASS] README found (README.md)
[PASS] License file found (LICENSE)
[PASS] .gitignore found
[PASS] GitHub Actions configuration found
[INFO] Project type: Go
[WARN] 2 files exceed 10.0 MiB
       assets/demo.mp4 (48.2 MiB)
       data/dump.sql (12.7 MiB)

Summary: 5 passed, 1 warning, 1 info (scanned in 4ms)
```

Colors are added to the status labels only when writing to an interactive terminal. They are off when `NO_COLOR` is set, when `TERM=dumb`, when output is piped or redirected, and on Windows. The report is complete without them.

### JSON output

`--format json` prints a single JSON document to stdout; errors go to stderr. See [docs/json-output.md](docs/json-output.md) for the schema and compatibility promise.

### Exit codes

| Code | Meaning |
|---|---|
| `0` | The scan completed with no warnings. |
| `1` | The scan completed and found at least one warning. |
| `2` | Invalid arguments or a fatal error (for example, the path does not exist). Nothing is written to stdout in this case. |

Findings about the repository (a missing README) are warnings, not errors, so they never produce exit code 2. Paths that could not be read are reported as INFO and do **not** change the exit code; check `unreadable_paths` in the JSON output if you need to treat them strictly.

## Large-file scan and exclusions

The scan walks the directory tree and compares each regular file's size with the threshold (strictly greater than: a file of exactly 10 MiB is not reported). It reads only file metadata, never contents.

- `.git` is always skipped. By default `node_modules`, `vendor`, `target`, `dist` and `build` are skipped too.
- Exclusions match a directory **name** exactly (case-sensitive) at any depth. There are no globs or paths.
- The directory you point adag-lens at is never skipped, even if it is called `build`.
- Symbolic links are never followed and never reported.
- Directories or files that cannot be read are skipped and listed in the report.

## Limitations

- **`.gitignore` is not read.** Untracked and ignored files are scanned like any other, so a large ignored file (for example a compiled binary that is not in a default-excluded directory) will be reported.
- **Exclusions are by directory name only.** No patterns, no paths.
- **It scans exactly the path you give.** If you point it at a subdirectory of a repository, repository-level files such as the README are looked for in that subdirectory. The report shows the Git root and a note when this happens.
- **Only GitHub Actions is detected** as CI, so a repository using another CI service gets an INFO line, not a PASS.
- **Existence checks only.** An empty README counts as a README.
- **File sizes are apparent sizes**, not disk usage; sparse or compressed files may differ.
- **File names that are not valid UTF-8** are shown with a Unicode replacement character, so such a name cannot be used to open the file.
- **Git detection is simple:** it looks for a `.git` entry in the path or a parent. Bare repositories are not detected, and the contents of a `.git` file (worktrees, submodules) are not parsed.
- **Project type detection** only looks at the five marker files listed above, in the top-level directory.
- **Platforms:** the test suite runs in CI on Linux, macOS and Windows, but the tool is developed on Linux and has had far less real-world use on the other two.

## Roadmap

**Implemented in 0.1.0:** everything described above.

**Planned (not implemented):**

- GitHub Actions integration (running adag-lens in a workflow)
- Configurable checks
- Markdown report output
- More project-type detectors
- Repository comparison
- Optional Git metadata checks
- More configurable directory exclusions

## Development

```bash
make check    # gofmt check, go vet, tests
make build    # build ./adag-lens
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for how the code is organized and how to add a check.

## License

[MIT](LICENSE)
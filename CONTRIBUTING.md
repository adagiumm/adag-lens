# Contributing to adag-lens

Thanks for your interest! adag-lens is a small project with a deliberately narrow scope: basic repository hygiene and discoverability. It is not a security scanner or a general linter. Ideas that fit that scope are very welcome.

## Getting started

You need a recent Go toolchain (see the `go` line in `go.mod`).

```bash
git clone https://github.com/adagiumm/adag-lens.git
cd adag-lens
make check     # gofmt check, go vet, go test
make build     # builds ./adag-lens
```

`make` is usually not installed on Windows. There, run the same commands directly:

```bash
gofmt -l .
go vet ./...
go test ./...
go build -o adag-lens.exe ./cmd/adag-lens
```

`gofmt -l .` must print nothing. CI runs these checks on Linux, macOS and Windows.

## How the code is organized

| Path | Responsibility |
|---|---|
| `cmd/adag-lens` | `main`: three lines that call `cli.Run` and exit. |
| `internal/cli` | Flag parsing, path validation, exit codes. |
| `internal/scan` | All scanning logic. Returns a `Result`; never prints. |
| `internal/report` | Renders a `Result` as text or JSON. Never touches the filesystem. |
| `internal/version` | Build version, set at link time. |

The key rule: **`scan` and `report` do not know about each other's concerns.** Findings are data in the `Result`. An `error` is reserved for "the scan could not run".

## Adding a check

1. Write a function `checkSomething(...) Check` in `internal/scan/checks.go` (or a new file for larger checks). Give it a stable `ID`.
2. Add it to the list in `scan.Run`, in `internal/scan/scan.go`. The order of that list is the order of the report.
3. Decide its status on failure and why: `WARN` means "worth fixing" and makes the tool exit with code 1; `INFO` is neutral.
4. Add table-driven tests in `internal/scan`, using temporary directories (`t.TempDir()`). Do not depend on files outside the test.
5. Update `docs/json-output.md` (the list of check IDs) and the table in `README.md`.

## Guidelines

- Keep it simple and idiomatic. Prefer a plain function over a new abstraction, and the standard library over a new dependency.
- Tests are expected with every change. The existing tests show the style; table-driven tests are the norm.
- The tool must never modify the repository it scans, print file contents, or make network requests.
- Output must be deterministic: stable ordering, no timestamps in JSON.
- Anything printed that comes from the scanned repository (file names, for example) is untrusted. The text report strips control characters; keep that in mind when adding output.
- Do not describe a planned feature as implemented in the docs.
- Write commit messages in the imperative mood ("Add X", "Fix Y").

## Reporting bugs and proposing features

Open an issue with the command you ran, the output you got, the output you expected, your operating system, and the result of `adag-lens --version`. For a feature, please explain the problem it solves and how it fits the project's scope.
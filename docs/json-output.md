# JSON output

`adag-lens --format json [path]` prints one JSON document to stdout.
Diagnostics and fatal errors go to stderr, and stdout stays empty when the
scan cannot run (exit code 2).

## Example

```json
{
  "schema_version": 1,
  "repository": {
    "path": "/home/me/project",
    "git_root": "/home/me/project",
    "project_types": ["Go"]
  },
  "checks": [
    {
      "id": "large-files",
      "status": "warn",
      "message": "1 file exceeds 10.0 MiB",
      "details": ["big.bin (11.0 MiB)"]
    }
  ],
  "large_files": [{ "path": "big.bin", "size_bytes": 11534336 }],
  "unreadable_paths": [],
  "summary": { "pass": 4, "warn": 1, "info": 2 }
}
```

## Fields

| Field | Type | Meaning |
|---|---|---|
| `schema_version` | number | Layout version, currently `1`. |
| `repository.path` | string | Absolute path that was scanned. |
| `repository.git_root` | string or `null` | Top of the enclosing Git working tree, or `null` if none was found. |
| `repository.project_types` | array of strings | Detected project types, in a fixed order. May be empty. |
| `checks[].id` | string | Stable check identifier: `git`, `readme`, `license`, `gitignore`, `ci`, `project-type`, `large-files`. |
| `checks[].status` | string | `pass`, `warn` or `info`. |
| `checks[].message` | string | Human-readable summary. **Wording may change; scripts should use `id` and `status`.** |
| `checks[].details` | array of strings | Extra lines, such as file names. May be empty. |
| `large_files[].path` | string | Path relative to `repository.path`, with forward slashes. Largest first. |
| `large_files[].size_bytes` | number | File size in bytes. |
| `unreadable_paths` | array of strings | Paths the scan could not read, relative to `repository.path`. |
| `summary.pass`, `.warn`, `.info` | number | Count of checks per status. |

Lists are always arrays, never `null`.

## Compatibility

- New fields may be added without changing `schema_version`, so consumers should ignore fields they don't know.
- Renaming or removing a field, or changing its meaning, increments `schema_version`.
- Scan duration is deliberately not included: it differs on every run.
- File names that are not valid UTF-8 are emitted with the Unicode replacement character, a limitation of JSON.

## Exit codes

`0` no warnings, `1` at least one warning, `2` invalid arguments or fatal error.
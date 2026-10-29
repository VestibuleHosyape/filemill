# filemill

**filemill** is a single-binary, cross-platform CLI for batch file operations. It focuses on recursive scanning, flexible filters, safe defaults (dry-run for destructive actions), and composable subcommands.

[中文文档](README-CN.md) · [Dev requirements](DevRequirements.md)

## Why

Day-to-day engineering and ops work often means repeating the same file chores: hashing trees, renaming in bulk, finding duplicates, archiving logs, cleaning old artifacts. Existing tools are either too narrow or too awkward to maintain as ad-hoc scripts.

filemill aims to be one binary you can drop onto Windows, macOS, or Linux and drive with predictable flags.

## Status

Early development. **Milestones M1–M3** are done: walker + filters, concurrent `hash`, and safe `rename`.

| Milestone | Scope | Status |
|---|---|---|
| M1 | CLI skeleton + global flags + walker + `list` | Done |
| M2 | Worker pool + atomic writes + `hash` | Done |
| M3 | `rename` + dry-run / apply + reporting | Done |
| M4 | `dedupe` + `clean` | Planned |
| M5 | Config file, excludes, output formats | Planned |
| M6 | Tests, docs, cross-platform release (v0.1.0) | Planned |

## Features

### Available now

- Recursive directory walk (`filepath.WalkDir`)
- Filters: extension, filename glob, regex, size, modification time
- Bounded worker pool (`--workers`, default `min(NumCPU, 8)`)
- Atomic writes (temp file + rename) and per-file error isolation
- Progress summary: total / success / failed / elapsed
- Global flags: `--workers`, `--dry-run`, `--log-file`
- `list` — print matching files (path, size, mtime)
- `hash` — batch MD5 / SHA-1 / SHA-256 (`table` / `csv` / `json`)
- `rename` — prefix, suffix, sequence, regex replace (default dry-run)

### Planned (MVP / P0 remaining)

- `dedupe` — size + hash duplicate detection (`--delete` optional)
- Rule-based `clean` (force dry-run confirmation)

### Later (P1 / P2)

- `compress` / `extract`
- `.filemill.yaml` config, `--exclude`, unified `--output`
- Resume / checkpoint, encoding convert, watch mode, plugins, TUI

## Requirements

- Go **1.27+** (see `go.mod`)

## Install

```bash
git clone <repo-url> filemill
cd filemill
go build -o filemill .
```

On Windows:

```powershell
go build -o filemill.exe .
```

## Quick start

```bash
# List files
filemill list ./testdata
filemill list ./testdata --ext .txt
filemill list ./testdata --name "*.log"

# Hash files
filemill hash ./testdata --ext .txt --output json
filemill hash ./testdata --algo md5 --output csv --workers 4
filemill hash ./testdata --output json --out hashes.json

# Rename (default dry-run; use --apply / --yes to execute)
filemill rename ./testdata --prefix new_
filemill rename ./testdata --prefix new_ --ext .txt --apply
filemill rename ./testdata --find "^old_" --repl "new_" --yes
filemill rename ./testdata --suffix _bak --seq --seq-start 1 --seq-width 3
```

### Global flags

| Flag | Description |
|---|---|
| `--workers int` | Concurrent workers (default `min(NumCPU, 8)`) |
| `--dry-run` | Preview actions without changing files |
| `--log-file path` | Also write structured logs to a file |

### Shared filter flags (`list` / `hash` / `rename`)

| Flag | Description |
|---|---|
| `--ext` | Extension filter (repeatable / comma-separated) |
| `--name` | Filename glob (`filepath.Match`) |
| `--regex` | Filename regular expression (which files to include) |
| `--min-size` / `--max-size` | Size bounds in bytes |
| `--after` / `--before` | mtime filters (`2006-01-02` or RFC3339) |

### `hash` flags

| Flag | Description |
|---|---|
| `--algo` | `md5` / `sha1` / `sha256` (default `sha256`) |
| `--output` | `table` / `csv` / `json` (default `table`) |
| `--out` | Atomically write results to a file instead of stdout |

### `rename` flags

| Flag | Description |
|---|---|
| `--prefix` | Prepend to the basename |
| `--suffix` | Insert before the extension |
| `--find` / `--repl` | Regex replace on the basename (`$1` groups supported) |
| `--seq` | Append a sequence number before the extension |
| `--seq-start` / `--seq-width` | Sequence start (default 1) and zero-pad width (default 3) |
| `--apply` / `--yes` | Actually perform renames (default is dry-run) |

Rename rules stack in order: regex → prefix → suffix → sequence. Global `--dry-run` overrides `--apply` / `--yes`.

## Safety

Destructive operations (`rename`, and upcoming delete / `clean`) **default to dry-run**. Real changes require `--apply` or `--yes`. Existing rename targets are not overwritten; conflicts are reported and other files continue.

## Project layout

```text
filemill/
├── main.go
├── cmd/                   # Cobra commands: list, hash, rename
├── internal/
│   ├── walker/            # Recursive walk + filters
│   ├── worker/            # Bounded worker pool
│   ├── fsutil/            # Atomic write, safe rename
│   ├── hashutil/          # File hashing
│   ├── renameutil/        # Rename planning / rules
│   └── report/            # Progress summary
├── testdata/
├── DevRequirements.md
├── README.md
└── README-CN.md
```

## Development

```bash
go test ./...
go build -o filemill .
```

Core logic lives under `internal/` so it stays testable without the CLI layer.

## License

See [LICENSE](LICENSE).

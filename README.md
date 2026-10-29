# filemill

**filemill** is a single-binary, cross-platform CLI for batch file operations. It focuses on recursive scanning, flexible filters, safe defaults (dry-run for destructive actions), and composable subcommands.

[中文文档](README-CN.md) · [Dev requirements](DevRequirements.md)

## Why

Day-to-day engineering and ops work often means repeating the same file chores: hashing trees, renaming in bulk, finding duplicates, archiving logs, cleaning old artifacts. Existing tools are either too narrow or too awkward to maintain as ad-hoc scripts.

filemill aims to be one binary you can drop onto Windows, macOS, or Linux and drive with predictable flags.

## Status

Early development. **Milestone M1** is done: Cobra CLI skeleton, global flags, recursive walker with filters, and a `list` command.

| Milestone | Scope | Status |
|---|---|---|
| M1 | CLI skeleton + global flags + walker + `list` | Done |
| M2 | Worker pool + atomic writes + `hash` | Planned |
| M3 | `rename` + dry-run / apply + reporting | Planned |
| M4 | `dedupe` + `clean` | Planned |
| M5 | Config file, excludes, output formats | Planned |
| M6 | Tests, docs, cross-platform release (v0.1.0) | Planned |

## Features

### Available now

- Recursive directory walk (`filepath.WalkDir`)
- Filters: extension, filename glob, regex, size, modification time
- Global flags: `--workers`, `--dry-run`, `--log-file`
- `list` — print matching files (path, size, mtime)

### Planned (MVP / P0)

- Concurrent processing with a bounded worker pool (`--workers`, default `min(NumCPU, 8)`)
- Atomic writes (temp file + `os.Rename`)
- Per-file error isolation (log and continue)
- Progress summary: total / success / failed / elapsed
- `hash` — batch MD5 / SHA-1 / SHA-256 (CSV / JSON)
- `rename` — prefix, suffix, sequence, regex replace (default dry-run)
- `dedupe` — size + hash duplicate detection (`--delete` optional)

### Later (P1 / P2)

- `compress` / `extract`, rule-based `clean`
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
# List all files under ./testdata
filemill list ./testdata

# Filter by extension
filemill list ./testdata --ext .txt

# Filter by filename glob
filemill list ./testdata --name "*.log"

# Filter by regex + size
filemill list ./testdata --regex "^app" --min-size 1
```

### Global flags

| Flag | Description |
|---|---|
| `--workers int` | Concurrent workers (default `min(NumCPU, 8)`) |
| `--dry-run` | Preview actions without changing files |
| `--log-file path` | Also write structured logs to a file |

### `list` flags

| Flag | Description |
|---|---|
| `--ext` | Extension filter (repeatable / comma-separated) |
| `--name` | Filename glob (`filepath.Match`) |
| `--regex` | Filename regular expression |
| `--min-size` / `--max-size` | Size bounds in bytes |
| `--after` / `--before` | mtime filters (`2006-01-02` or RFC3339) |

## Safety

Destructive operations (`rename`, delete, `clean`, …) will **default to dry-run**. Real changes require an explicit confirmation flag such as `--apply` or `--yes` (wired as those commands land).

## Project layout

```text
filemill/
├── main.go
├── cmd/                 # Cobra commands (root, list, …)
├── internal/
│   └── walker/          # Recursive walk + filters
├── testdata/            # Sample files for local checks
├── DevRequirements.md   # Product / engineering requirements
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

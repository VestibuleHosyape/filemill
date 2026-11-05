# filemill

**filemill** is a single-binary, cross-platform CLI for batch file operations. It focuses on recursive scanning, flexible filters, safe defaults (dry-run for destructive actions), and composable subcommands.

[中文文档](README-CN.md) · [Dev requirements](DevRequirements.md)

## Why

Day-to-day engineering and ops work often means repeating the same file chores: hashing trees, renaming in bulk, finding duplicates, archiving logs, cleaning old artifacts. Existing tools are either too narrow or too awkward to maintain as ad-hoc scripts.

filemill aims to be one binary you can drop onto Windows, macOS, or Linux and drive with predictable flags.

## Status

Early development. `list`, concurrent `hash`, safe-default `rename`, and `dedupe` with verified, explicitly confirmed deletion are implemented. Remote cross-platform CI still needs to pass before release.

| Milestone | Scope | Status |
|---|---|---|
| M1 | CLI skeleton + global flags + walker + `list` | Done |
| M2 | Worker pool + atomic writes + `hash` | Done |
| M3 | `rename` + dry-run / apply + reporting | Done |
| M4 | `dedupe`; defer `clean` to P1 | `dedupe` implemented |
| M5 | Config file, excludes, output formats | Excludes implemented; config and unified output pending |
| M6 | Tests, docs, cross-platform release (v0.1.0) | Tests, CI and packaging added; remote validation and publication pending |

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
- `dedupe` — size + SHA-256 detection, deterministic keeper selection, table / csv / json, verified deletion
- `--exclude` — file/directory patterns, pruning matching directory subtrees

### Later (P1 / P2)

- Rule-based `clean` (default dry-run)
- `compress` / `extract`
- `.filemill.yaml` config, unified `--output`
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

See [RELEASING.md](RELEASING.md) for packaging and performance checks. Build development binaries into `dist/` to avoid overwriting the old binary already in the repository.

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

### Shared filter flags (`list` / `hash` / `rename` / `dedupe`)

| Flag | Description |
|---|---|
| `--ext` | Extension filter (repeatable / comma-separated) |
| `--name` | Filename glob (`filepath.Match`) |
| `--regex` | Filename regular expression (which files to include) |
| `--min-size` / `--max-size` | Size bounds in bytes |
| `--after` / `--before` | mtime filters (`2006-01-02` or RFC3339) |
| `--exclude` | Repeatable glob against basenames or root-relative paths using `/`; matching directories are pruned, no `**`, no implicit excludes |

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

### Duplicate detection and deletion

```bash
filemill dedupe ./data --exclude .git --exclude node_modules --output json
filemill dedupe ./data --delete                 # Preview only
filemill dedupe ./data --delete --apply         # Execute
filemill dedupe ./data --delete --yes --dry-run # Still preview only
```

Only equal-size candidates are hashed using SHA-256. Each group keeps its first path in lexical string order. Hardlink aliases of that keeper are skipped; empty files participate. Symlinks are not followed, including a root path that is itself a symlink. `reclaimable_bytes` counts logical duplicate bytes, not guaranteed physical disk savings (hardlinks, sparse files and compression affect actual allocation).

`--apply` / `--yes` require `--delete`. Any scan/hash error blocks all deletion in that run. During deletion, a failed item does not prevent other items from being processed; the final exit status is nonzero. Before deletion, both files are rechecked for identity, size, mtime, SHA-256 and byte equality. **Stop concurrent writes in the target tree first**: portable pathname deletion cannot eliminate replacement races between the final check and removal. Deletion has no undo.

JSON includes `groups`, `errors`, `scanned`, `hashed`, `reclaimable_bytes`, `mode` and `actions`. CSV has one row per duplicate with keeper path, size, hash, status and error. Summaries go to stderr.

## Safety

Destructive operations (`rename`, `dedupe --delete`, and future `clean`) **default to dry-run**. Real changes require `--apply` or `--yes`. Existing rename targets are not overwritten; conflicts are reported and other files continue.

Scanning continues over accessible files after filesystem errors and returns a nonzero status if scanning or processing failed. Output replacement never deletes the existing destination before attempting the rename. Atomic visibility on non-Unix systems depends on the OS/filesystem.

## Project layout

```text
filemill/
├── main.go
├── cmd/                   # Cobra commands: list, hash, rename, dedupe
├── internal/
│   ├── walker/            # Recursive walk + filters
│   ├── worker/            # Bounded worker pool
│   ├── fsutil/            # Atomic write, safe rename
│   ├── hashutil/          # File hashing
│   ├── dedupe/            # Duplicate planning and verified deletion
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

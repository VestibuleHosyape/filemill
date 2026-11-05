# filemill

**filemill** 是一个单二进制、跨平台的文件批量处理 CLI。面向递归扫描、灵活过滤、安全默认（破坏性操作默认 dry-run），以及可组合的子命令。

[English](README.md) · [开发需求文档](DevRequirements.md)

## 为什么做

开发和运维里经常要重复处理大量文件：批量算哈希、批量重命名、查重、归档日志、清理旧产物。现成工具要么功能太单一，要么只能靠零散脚本硬撑。

filemill 的目标是提供一个可在 Windows / macOS / Linux 上直接使用的单二进制工具，用统一、可预期的 flag 驱动常见批量操作。

## 当前状态

项目处于早期开发阶段。已实现 `list`、并发 `hash`、安全默认的 `rename`，以及 `dedupe` 查重与显式确认删除。发布前仍需确认远程跨平台 CI 结果。

| 里程碑 | 内容 | 状态 |
|---|---|---|
| M1 | CLI 骨架 + 全局 flag + walker + `list` | 已完成 |
| M2 | 并发池 + 原子写入 + `hash` | 已完成 |
| M3 | `rename` + dry-run / apply + 报告 | 已完成 |
| M4 | `dedupe`；`clean` 延后至 P1 | `dedupe` 已实现 |
| M5 | 配置文件、排除规则、输出格式 | 排除规则已实现；配置和统一输出待做 |
| M6 | 单测、文档、跨平台发布（v0.1.0） | 测试、CI 和打包脚本已添加；待远程验证及发布 |

## 功能

### 现已可用

- 递归目录遍历（`filepath.WalkDir`）
- 过滤：扩展名、文件名 glob、正则、大小、修改时间
- 带限流的并发 worker 池（`--workers`，默认 `min(NumCPU, 8)`）
- 原子写入（临时文件 + rename）与单文件错误隔离
- 进度汇总：总数 / 成功 / 失败 / 耗时
- 全局 flag：`--workers`、`--dry-run`、`--log-file`
- `list` — 打印匹配文件（路径、大小、修改时间）
- `hash` — 批量 MD5 / SHA-1 / SHA-256（`table` / `csv` / `json`）
- `rename` — 前缀、后缀、序号、正则替换（默认 dry-run）
- `dedupe` — 大小 + SHA-256 查重，稳定保留规则，支持 table / csv / json 和删除前复核
- `--exclude` — 排除文件与目录，匹配目录时直接跳过子树

### 后续（P1 / P2）

- 按规则 `clean`（默认 dry-run）
- `compress` / `extract`
- `.filemill.yaml` 配置、统一 `--output`
- 断点续跑、编码转换、目录监听、插件、TUI

## 环境要求

- Go **1.27+**（见 `go.mod`）

## 安装

```bash
git clone <repo-url> filemill
cd filemill
go build -o filemill .
```

Windows：

```powershell
go build -o filemill.exe .
```

发布打包及性能验证见 [RELEASING.md](RELEASING.md)。本地开发建议构建至 `dist/`，避免覆盖仓库中已有的旧二进制。

## 快速开始

```bash
# 列出文件
filemill list ./testdata
filemill list ./testdata --ext .txt
filemill list ./testdata --name "*.log"

# 计算哈希
filemill hash ./testdata --ext .txt --output json
filemill hash ./testdata --algo md5 --output csv --workers 4
filemill hash ./testdata --output json --out hashes.json

# 重命名（默认 dry-run；加 --apply / --yes 才执行）
filemill rename ./testdata --prefix new_
filemill rename ./testdata --prefix new_ --ext .txt --apply
filemill rename ./testdata --find "^old_" --repl "new_" --yes
filemill rename ./testdata --suffix _bak --seq --seq-start 1 --seq-width 3
```

### 全局 flag

| Flag | 说明 |
|---|---|
| `--workers int` | 并发 worker 数（默认 `min(NumCPU, 8)`） |
| `--dry-run` | 试运行：只预览，不改动文件 |
| `--log-file path` | 同时将结构化日志写入文件 |

### 共用过滤 flag（`list` / `hash` / `rename` / `dedupe`）

| Flag | 说明 |
|---|---|
| `--ext` | 扩展名过滤（可重复 / 逗号分隔） |
| `--name` | 文件名 glob（`filepath.Match`） |
| `--regex` | 文件名正则（筛选要处理的文件） |
| `--min-size` / `--max-size` | 大小上下限（字节） |
| `--after` / `--before` | 修改时间过滤（`2006-01-02` 或 RFC3339） |
| `--exclude` | 可重复的 glob：按文件/目录名，或使用 `/` 的根目录相对路径匹配；不支持 `**`，没有隐式默认排除 |

### `hash` flag

| Flag | 说明 |
|---|---|
| `--algo` | `md5` / `sha1` / `sha256`（默认 `sha256`） |
| `--output` | `table` / `csv` / `json`（默认 `table`） |
| `--out` | 将结果原子写入文件（默认打印到标准输出） |

### `rename` flag

| Flag | 说明 |
|---|---|
| `--prefix` | 添加到文件名前的前缀 |
| `--suffix` | 插入到扩展名之前的后缀 |
| `--find` / `--repl` | 对文件名做正则替换（支持 `$1` 分组） |
| `--seq` | 在扩展名之前追加序号 |
| `--seq-start` / `--seq-width` | 序号起始值（默认 1）与零填充宽度（默认 3） |
| `--apply` / `--yes` | 真正执行重命名（默认为 dry-run） |

规则叠加顺序：正则 → 前缀 → 后缀 → 序号。全局 `--dry-run` 会覆盖 `--apply` / `--yes`。

### 查重与删除

```bash
filemill dedupe ./data --exclude .git --exclude node_modules --output json
filemill dedupe ./data --delete                 # 仅预览删除计划
filemill dedupe ./data --delete --apply         # 确认执行
filemill dedupe ./data --delete --yes --dry-run # 仍然只预览
```

只对大小相同的候选文件计算 SHA-256。每组按路径字符串升序保留第一份，跳过保留文件的硬链接别名；空文件也参与查重。扫描不会跟随符号链接，包括直接传入的符号链接路径。`reclaimable_bytes` 是重复项的逻辑字节总数，不保证实际释放的磁盘空间（硬链接、稀疏文件和文件系统压缩会影响它）。

`--apply` / `--yes` 必须与 `--delete` 一起使用。扫描或哈希发生任何错误时，本次删除整体禁止；删除阶段单项失败会继续处理其他项，最终返回非零退出码。删除前重新检查保留项与重复项的文件身份、大小、修改时间、SHA-256，并逐字节比较。**请先停止目标目录的并发写入**：跨平台的路径删除无法彻底消除最后检查与删除之间的文件替换竞态。删除无撤销功能。

JSON 包含 `groups`、`errors`、`scanned`、`hashed`、`reclaimable_bytes`、`mode`、`actions`；CSV 每行表示一个重复项及其保留路径、大小、哈希、状态和错误。进度汇总输出到 stderr。

## 安全约定

破坏性操作（`rename`、`dedupe --delete`，以及后续的 `clean`）**默认 dry-run**。真正改动需要 `--apply` 或 `--yes`。重命名不会覆盖已存在目标；冲突会明确报错，其余文件继续处理。

可访问的文件会在其他文件扫描失败后继续处理；扫描或处理有失败时返回非零退出码。写入结果先完成临时文件再替换目标，替换失败不会先删除旧目标；非 Unix 平台的原子可见性取决于操作系统和文件系统。

## 项目结构

```text
filemill/
├── main.go
├── cmd/                   # Cobra 命令：list、hash、rename、dedupe
├── internal/
│   ├── walker/            # 递归遍历 + 过滤
│   ├── worker/            # 并发池
│   ├── fsutil/            # 原子写入、安全重命名
│   ├── hashutil/          # 文件哈希
│   ├── dedupe/            # 查重计划与删除前复核
│   ├── renameutil/        # 重命名规则与计划
│   └── report/            # 进度汇总
├── testdata/
├── DevRequirements.md
├── README.md
└── README-CN.md
```

## 开发

```bash
go test ./...
go build -o filemill .
```

核心逻辑放在 `internal/`，与 CLI 解耦，便于单测。

## 许可证

见 [LICENSE](LICENSE)。

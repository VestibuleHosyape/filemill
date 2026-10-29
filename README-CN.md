# filemill

**filemill** 是一个单二进制、跨平台的文件批量处理 CLI。面向递归扫描、灵活过滤、安全默认（破坏性操作默认 dry-run），以及可组合的子命令。

[English](README.md) · [开发需求文档](DevRequirements.md)

## 为什么做

开发和运维里经常要重复处理大量文件：批量算哈希、批量重命名、查重、归档日志、清理旧产物。现成工具要么功能太单一，要么只能靠零散脚本硬撑。

filemill 的目标是提供一个可在 Windows / macOS / Linux 上直接使用的单二进制工具，用统一、可预期的 flag 驱动常见批量操作。

## 当前状态

项目处于早期开发阶段。**里程碑 M1～M3** 已完成：walker + 过滤、并发 `hash`、以及安全的 `rename`。

| 里程碑 | 内容 | 状态 |
|---|---|---|
| M1 | CLI 骨架 + 全局 flag + walker + `list` | 已完成 |
| M2 | 并发池 + 原子写入 + `hash` | 已完成 |
| M3 | `rename` + dry-run / apply + 报告 | 已完成 |
| M4 | `dedupe` + `clean` | 计划中 |
| M5 | 配置文件、排除规则、输出格式 | 计划中 |
| M6 | 单测、文档、跨平台发布（v0.1.0） | 计划中 |

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

### 计划中（剩余 MVP / P0）

- `dedupe` — 按大小 + 哈希查重（可选 `--delete`）
- 按规则 `clean`（强制 dry-run 确认）

### 后续（P1 / P2）

- `compress` / `extract`
- `.filemill.yaml` 配置、`--exclude`、统一 `--output`
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

### 共用过滤 flag（`list` / `hash` / `rename`）

| Flag | 说明 |
|---|---|
| `--ext` | 扩展名过滤（可重复 / 逗号分隔） |
| `--name` | 文件名 glob（`filepath.Match`） |
| `--regex` | 文件名正则（筛选要处理的文件） |
| `--min-size` / `--max-size` | 大小上下限（字节） |
| `--after` / `--before` | 修改时间过滤（`2006-01-02` 或 RFC3339） |

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

## 安全约定

破坏性操作（`rename`，以及后续的删除 / `clean`）**默认 dry-run**。真正改动需要 `--apply` 或 `--yes`。重命名不会覆盖已存在目标；冲突会明确报错，其余文件继续处理。

## 项目结构

```text
filemill/
├── main.go
├── cmd/                   # Cobra 命令：list、hash、rename
├── internal/
│   ├── walker/            # 递归遍历 + 过滤
│   ├── worker/            # 并发池
│   ├── fsutil/            # 原子写入、安全重命名
│   ├── hashutil/          # 文件哈希
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

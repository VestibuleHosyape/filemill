# filemill

**filemill** 是一个单二进制、跨平台的文件批量处理 CLI。面向递归扫描、灵活过滤、安全默认（破坏性操作默认 dry-run），以及可组合的子命令。

[English](README.md) · [开发需求文档](DevRequirements.md)

## 为什么做

开发和运维里经常要重复处理大量文件：批量算哈希、批量重命名、查重、归档日志、清理旧产物。现成工具要么功能太单一，要么只能靠零散脚本硬撑。

filemill 的目标是提供一个可在 Windows / macOS / Linux 上直接使用的单二进制工具，用统一、可预期的 flag 驱动常见批量操作。

## 当前状态

项目处于早期开发阶段。**里程碑 M1** 已完成：Cobra CLI 骨架、全局 flag、带过滤的递归 walker，以及 `list` 子命令。

| 里程碑 | 内容 | 状态 |
|---|---|---|
| M1 | CLI 骨架 + 全局 flag + walker + `list` | 已完成 |
| M2 | 并发池 + 原子写入 + `hash` | 计划中 |
| M3 | `rename` + dry-run / apply + 报告 | 计划中 |
| M4 | `dedupe` + `clean` | 计划中 |
| M5 | 配置文件、排除规则、输出格式 | 计划中 |
| M6 | 单测、文档、跨平台发布（v0.1.0） | 计划中 |

## 功能

### 现已可用

- 递归目录遍历（`filepath.WalkDir`）
- 过滤：扩展名、文件名 glob、正则、大小、修改时间
- 全局 flag：`--workers`、`--dry-run`、`--log-file`
- `list` — 打印匹配文件（路径、大小、修改时间）

### 计划中（MVP / P0）

- 带缓冲 channel 的并发 worker 池（`--workers`，默认 `min(NumCPU, 8)`）
- 原子写入（先写临时文件，再 `os.Rename`）
- 单文件错误隔离（记录日志并继续）
- 进度汇总：总数 / 成功 / 失败 / 耗时
- `hash` — 批量 MD5 / SHA-1 / SHA-256（CSV / JSON）
- `rename` — 前缀、后缀、序号、正则替换（默认 dry-run）
- `dedupe` — 按大小 + 哈希查重（可选 `--delete`）

### 后续（P1 / P2）

- `compress` / `extract`、按规则 `clean`
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
# 列出 ./testdata 下所有文件
filemill list ./testdata

# 按扩展名过滤
filemill list ./testdata --ext .txt

# 按文件名 glob 过滤
filemill list ./testdata --name "*.log"

# 按正则 + 大小过滤
filemill list ./testdata --regex "^app" --min-size 1
```

### 全局 flag

| Flag | 说明 |
|---|---|
| `--workers int` | 并发 worker 数（默认 `min(NumCPU, 8)`） |
| `--dry-run` | 试运行：只预览，不改动文件 |
| `--log-file path` | 同时将结构化日志写入文件 |

### `list` flag

| Flag | 说明 |
|---|---|
| `--ext` | 扩展名过滤（可重复 / 逗号分隔） |
| `--name` | 文件名 glob（`filepath.Match`） |
| `--regex` | 文件名正则 |
| `--min-size` / `--max-size` | 大小上下限（字节） |
| `--after` / `--before` | 修改时间过滤（`2006-01-02` 或 RFC3339） |

## 安全约定

破坏性操作（`rename`、删除、`clean` 等）将**默认 dry-run**。真正改动文件需要显式确认，例如 `--apply` 或 `--yes`（随对应子命令落地）。

## 项目结构

```text
filemill/
├── main.go
├── cmd/                 # Cobra 命令（root、list、…）
├── internal/
│   └── walker/          # 递归遍历 + 过滤
├── testdata/            # 本地验证样例文件
├── DevRequirements.md   # 产品 / 工程需求
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

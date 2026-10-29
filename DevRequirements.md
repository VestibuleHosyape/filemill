# filemill 开发需求文档

- 文档版本：v0.1
- 日期：2026-10-29
- 状态：草案
- 项目类型：Go CLI 工具
- 模块名建议：`github.com/你的用户名/filemill`

## 1. 项目背景

日常开发和运维中经常需要对大量文件做重复性操作：批量压缩日志、批量重命名、批量转换编码、查找重复文件、按规则归档等。现成工具要么功能单一，要么脚本难维护。

filemill 的目标是提供一个**单二进制、跨平台、可组合子命令**的文件批量处理工具。

## 2. 目标用户

- 开发者：清理构建产物、批量处理日志、整理素材
- 运维：归档日志、批量校验文件
- 普通用户：批量重命名、查找重复文件

## 3. 功能需求

### 3.1 优先级定义

- P0：MVP 必须实现
- P1：应该实现
- P2：可选增强

### 3.2 P0：必须实现

| 编号 | 功能 | 说明 |
|---|---|---|
| F-01 | 递归遍历 | 基于 `filepath.WalkDir` 递归扫描目录 |
| F-02 | 过滤规则 | 按扩展名、文件名 glob、正则、大小、修改时间过滤 |
| F-03 | 并发处理 | 带缓冲 channel 限流，`--workers` 可配置，默认 `min(NumCPU, 8)` |
| F-04 | 原子写入 | 输出先写临时文件，成功后 `os.Rename`，避免半成品 |
| F-05 | 错误隔离 | 单文件失败记录日志并继续，不中断整体任务 |
| F-06 | 试运行模式 | `--dry-run` 只打印将要执行的操作，不实际改动 |
| F-07 | 进度反馈 | 处理总数、成功数、失败数、耗时 |
| F-08 | 子命令 `hash` | 批量计算文件哈希（md5/sha1/sha256），支持输出 CSV/JSON |
| F-09 | 子命令 `rename` | 按规则批量重命名（前缀、后缀、序号、正则替换），默认 dry-run |
| F-10 | 子命令 `dedupe` | 按大小 + 哈希查找重复文件，支持 `--delete` 删除重复项 |

### 3.3 P1：应该实现

| 编号 | 功能 | 说明 |
|---|---|---|
| F-11 | 子命令 `compress` | 批量压缩为 zip/tar.gz，支持按目录分组 |
| F-12 | 子命令 `extract` | 批量解压 |
| F-13 | 子命令 `clean` | 按规则清理（如删除 7 天前的 `.log`），强制 dry-run 确认 |
| F-14 | 配置文件 | 支持 `.filemill.yaml` 定义常用规则集 |
| F-15 | 排除规则 | `--exclude` 支持隐藏文件、`.git`、`node_modules` 等 |
| F-16 | 输出格式 | `--output json/csv/table`，便于管道处理 |
| F-17 | 断点续跑 | 记录已处理文件清单，中断后可跳过 |

### 3.4 P2：可选增强

| 编号 | 功能 | 说明 |
|---|---|---|
| F-18 | 子命令 `convert` | 文本编码转换（UTF-8/GBK）、换行符统一 |
| F-19 | 子命令 `watch` | 监听目录变化，自动触发规则 |
| F-20 | 插件式规则 | 通过外部命令或脚本扩展处理逻辑 |
| F-21 | TUI 进度界面 | 用 Bubble Tea 展示实时进度 |

## 4. 非功能需求

- 跨平台：Windows / macOS / Linux，单二进制分发
- 安全性：所有破坏性操作（rename/delete/clean）默认 dry-run，需 `--yes` 或 `--apply` 才执行
- 可观测：结构化日志（`log/slog`），失败明细写入 `--log-file`
- 性能：10 万文件目录下遍历 + 哈希，内存占用不超过 200MB
- 可测试：核心逻辑与 CLI 解耦，`internal/fileutil` 单测覆盖率 ≥ 70%

## 5. 技术方案

- CLI 框架：`github.com/spf13/cobra`
- 配置：`github.com/spf13/viper`（P1）
- 日志：标准库 `log/slog`
- 并发：goroutine + 带缓冲 channel + `sync.WaitGroup`
- 哈希：标准库 `crypto/*`
- 压缩：标准库 `archive/zip`、`archive/tar`、`compress/gzip`
- 测试：标准库 `testing` + `testify`（可选）

## 6. 项目结构

```text
filemill/
├── main.go
├── cmd/
│   ├── root.go          # 全局 flag：--workers --dry-run --log-file
│   ├── hash.go
│   ├── rename.go
│   ├── dedupe.go
│   ├── compress.go
│   └── clean.go
├── internal/
│   ├── walker/          # 遍历 + 过滤
│   ├── worker/          # 并发池、限流
│   ├── fsutil/          # 原子写入、临时文件
│   ├── hashutil/
│   └── report/          # 进度与结果输出
└── testdata/
```

## 7. 验收标准

- `filemill hash ./testdata --ext .txt --output json` 输出合法 JSON，包含每个文件的哈希
- `filemill rename ./testdata --prefix new_ --dry-run` 只打印不修改
- 加 `--apply` 后文件被正确重命名，失败文件有明确错误信息
- `filemill dedupe ./testdata` 能识别内容相同但文件名不同的文件
- 处理 1 万个文件时并发可配置，进程不 panic，失败不中断
- `go test ./...` 全部通过

## 8. 开发里程碑

| 阶段 | 内容 | 产出 |
|---|---|---|
| M1 | Cobra 骨架 + 全局 flag + walker | 能遍历并打印文件列表 |
| M2 | worker 并发池 + 原子写入 + 错误隔离 | `hash` 子命令可用 |
| M3 | `rename` + dry-run + 报告 | 安全重命名 |
| M4 | `dedupe` + `clean` | 三个核心子命令完成 |
| M5 | 配置、排除规则、输出格式 | P1 完成 |
| M6 | 单测、文档、跨平台构建 | 可发布 v0.1.0 |

## 9. 开发约定

- 破坏性操作一律默认 dry-run
- 全局 flag 保持统一风格：`--dry-run`、`--yes`、`--output`、`--log-file`
- 先完成 M1～M2 跑通最小闭环，再迭代后续里程碑
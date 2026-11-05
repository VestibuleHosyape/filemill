# 验证与发布准备

## 本地验证

```powershell
go test -cover ./...
go vet ./...
go build -o dist/filemill.exe .
./dist/filemill.exe --version
./dist/filemill.exe dedupe ./testdata --output json
```

Linux/macOS 将二进制名改为 `dist/filemill`。有 C 编译器的环境执行 `go test -race ./...`。CI 在 Windows、Linux、macOS 上执行测试、vet 和构建，并在 Linux/macOS 上执行 race 检测。

CI 使用 [setup-go](https://github.com/actions/setup-go) 从 `go.mod` 读取版本，并通过 [upload-artifact](https://github.com/actions/upload-artifact) 保存打包产物。添加工作流不等于已经通过远程 CI，发布前需要查看实际运行结果。

## 本地打包

```powershell
go run ./tools/release -version v0.1.0-dev
```

产物位于 `dist/`：Windows、Linux、macOS 各提供 amd64 / arm64；Windows 为 ZIP，其余为 tar.gz，含可执行文件、双语 README 与 LICENSE。`SHA256SUMS` 可校验完整性，版本通过 `--version` 查看。打包器不会创建 tag、推送代码或发布 GitHub Release。正式发布时将版本参数换为 `v0.1.0`，先通过 CI 并实际验证各平台运行结果。

手动触发 CI 会创建开发版构建产物；推送 `v*` tag 会使用该 tag 作为版本打包。工作流仅上传 Actions artifact，不自动创建公开 Release。

`dist/` 与开发缓存已加入 `.gitignore`。仓库之前跟踪的根目录 `filemill.exe`、`.idea`、`.gai` 不会因添加 ignore 自动取消跟踪；本轮不删除这些既有文件，也不覆盖旧二进制。日常使用请运行新构建的 `dist/filemill.exe`。

## 十万文件基准

```powershell
$env:FILEMILL_BENCH_FILES = '100000'
go test ./internal/dedupe -run '^$' -bench BenchmarkTree -benchtime=1x -benchmem
Remove-Item Env:FILEMILL_BENCH_FILES
```

基准在临时目录创建十万个同内容小文件（18 字节），然后测量遍历、全部候选哈希和分组；准备与清理时间不计入基准时间。默认不设置环境变量时仅创建 1000 个文件。`B/op` 是累计分配量，不是峰值内存，不能直接用它判断 200 MB 要求。

2026-11-05 本地 Windows amd64 / Go 1.27.3 / Ryzen 5 7500F，8 workers，单次实测：十万文件处理约 **2.61 秒**，测试进程峰值工作集约 **137.63 MiB**。该结果覆盖小文件场景，不能代替大文件、长路径、不同磁盘或各操作系统的性能验证。

## 删除前提

先执行只读查重或 `--delete` 预览，确认保留路径。停止目录中的并发写入后再加 `--apply`。文件身份、元数据、哈希与字节比较可以拒绝已经变化的文件，但跨平台路径删除仍存在最后检查后的替换窗口；本工具不承诺对抗恶意并发修改，也没有撤销删除功能。

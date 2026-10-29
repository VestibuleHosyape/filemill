package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"

	"filemill/internal/fsutil"
	"filemill/internal/renameutil"
	"filemill/internal/report"
	"filemill/internal/walker"

	"github.com/spf13/cobra"
)

var (
	renameFilters  filterFlags
	renamePrefix   string
	renameSuffix   string
	renameFind     string
	renameRepl     string
	renameSeq      bool
	renameSeqStart int
	renameSeqWidth int
	renameApply    bool
	renameYes      bool
)

var renameCmd = &cobra.Command{
	Use:   "rename [path]",
	Short: "按规则批量重命名文件",
	Long: `递归扫描目录（默认当前目录），按前缀 / 后缀 / 序号 / 正则替换批量重命名。

安全约定：默认 dry-run，只打印计划，不改动文件。
加 --apply 或 --yes 后才会真正执行；若同时指定全局 --dry-run，仍只预览。

规则可叠加，应用顺序：正则替换 → 前缀 → 后缀（扩展名之前）→ 序号。`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRename,
}

func init() {
	rootCmd.AddCommand(renameCmd)
	renameFilters.register(renameCmd)

	renameCmd.Flags().StringVar(&renamePrefix, "prefix", "", "添加到文件名前的前缀")
	renameCmd.Flags().StringVar(&renameSuffix, "suffix", "", "插入到扩展名之前的后缀")
	renameCmd.Flags().StringVar(&renameFind, "find", "", "对文件名做正则查找（配合 --repl）")
	renameCmd.Flags().StringVar(&renameRepl, "repl", "", "正则替换内容，支持 $1 等分组引用")
	renameCmd.Flags().BoolVar(&renameSeq, "seq", false, "在扩展名之前追加序号")
	renameCmd.Flags().IntVar(&renameSeqStart, "seq-start", 1, "序号起始值")
	renameCmd.Flags().IntVar(&renameSeqWidth, "seq-width", 3, "序号数字宽度（零填充）")
	renameCmd.Flags().BoolVar(&renameApply, "apply", false, "真正执行重命名（默认为 dry-run）")
	renameCmd.Flags().BoolVar(&renameYes, "yes", false, "同 --apply，确认执行破坏性操作")
}

func runRename(cmd *cobra.Command, args []string) error {
	root := "."
	if len(args) == 1 {
		root = args[0]
	}

	opts := renameutil.Options{
		Prefix:   renamePrefix,
		Suffix:   renameSuffix,
		Find:     renameFind,
		Repl:     renameRepl,
		Seq:      renameSeq,
		SeqStart: renameSeqStart,
		SeqWidth: renameSeqWidth,
	}
	if !opts.HasRule() {
		return fmt.Errorf("请至少指定一种重命名规则：--prefix / --suffix / --find / --seq")
	}
	if renameFind == "" && renameRepl != "" {
		return fmt.Errorf("--repl 需要同时提供 --find")
	}

	walkOpts, err := renameFilters.options()
	if err != nil {
		return err
	}

	files, err := walker.Collect(root, walkOpts)
	if err != nil {
		return err
	}
	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.Path
	}
	sort.Strings(paths)

	mappings, err := renameutil.Plan(paths, opts)
	if err != nil {
		return err
	}

	execute := (renameApply || renameYes) && !DryRun()
	stats := &report.Stats{}
	stats.Start()

	for _, m := range mappings {
		stats.AddTotal(1)
		if m.Skipped {
			stats.IncSuccess()
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "skip\t%s\t(unchanged)\n", m.OldPath)
			continue
		}

		if !execute {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "dry-run\trename\t%s\t->\t%s\n", m.OldPath, m.NewPath)
			stats.IncSuccess()
			continue
		}

		if err := applyOneRename(m); err != nil {
			stats.IncFailed()
			slog.Error("rename failed", "from", m.OldPath, "to", m.NewPath, "err", err)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "error\t%s\t->\t%s\t%s\n", m.OldPath, m.NewPath, err.Error())
			continue
		}
		stats.IncSuccess()
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "renamed\t%s\t->\t%s\n", m.OldPath, m.NewPath)
	}

	mode := "dry-run"
	if execute {
		mode = "apply"
	}
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "mode=%s %s\n", mode, stats.Summary())
	return nil
}

func applyOneRename(m renameutil.Mapping) error {
	if _, err := os.Stat(m.OldPath); err != nil {
		return fmt.Errorf("源文件不可用: %w", err)
	}
	// Refuse clobbering a path that already exists and is not the source.
	if absOld, err1 := filepath.Abs(m.OldPath); err1 == nil {
		if absNew, err2 := filepath.Abs(m.NewPath); err2 == nil && absOld != absNew {
			if _, err := os.Stat(m.NewPath); err == nil {
				return fmt.Errorf("目标已存在: %s", m.NewPath)
			}
		}
	}
	return fsutil.RenameNoOverwrite(m.OldPath, m.NewPath)
}

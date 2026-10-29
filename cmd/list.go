package cmd

import (
	"fmt"
	"time"

	"filemill/internal/walker"

	"github.com/spf13/cobra"
)

var listFilters filterFlags

var listCmd = &cobra.Command{
	Use:   "list [path]",
	Short: "递归遍历并打印匹配的文件列表",
	Long: `递归扫描目录（默认当前目录），按扩展名、文件名 glob、正则、大小、修改时间过滤，
并将匹配文件逐行打印到标准输出。

该命令用于验证 walker 与全局 flag，也是后续子命令的扫描基础。`,
	Args: cobra.MaximumNArgs(1),
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listFilters.register(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	root := "."
	if len(args) == 1 {
		root = args[0]
	}

	opts, err := listFilters.options()
	if err != nil {
		return err
	}

	var count int
	err = walker.Walk(root, opts, func(fi walker.FileInfo) error {
		count++
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%d\t%s\n", fi.Path, fi.Size, fi.ModTime.Format(time.RFC3339))
		return err
	})
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(cmd.ErrOrStderr(), "listed %d file(s), workers=%d, dry-run=%v\n", count, Workers(), DryRun())
	return err
}

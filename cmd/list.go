package cmd

import (
	"fmt"
	"time"

	"filemill/internal/walker"

	"github.com/spf13/cobra"
)

var (
	listExts     []string
	listNameGlob string
	listRegex    string
	listMinSize  int64
	listMaxSize  int64
	listAfter    string
	listBefore   string
)

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

	listCmd.Flags().StringSliceVar(&listExts, "ext", nil, "按扩展名过滤，可重复或逗号分隔，如 --ext .txt,.log")
	listCmd.Flags().StringVar(&listNameGlob, "name", "", "按文件名 glob 过滤，如 --name '*.log'")
	listCmd.Flags().StringVar(&listRegex, "regex", "", "按文件名正则过滤")
	listCmd.Flags().Int64Var(&listMinSize, "min-size", 0, "最小文件大小（字节）")
	listCmd.Flags().Int64Var(&listMaxSize, "max-size", 0, "最大文件大小（字节）")
	listCmd.Flags().StringVar(&listAfter, "after", "", "仅包含此时间之后修改的文件，格式 2006-01-02 或 RFC3339")
	listCmd.Flags().StringVar(&listBefore, "before", "", "仅包含此时间之前修改的文件，格式 2006-01-02 或 RFC3339")
}

func runList(cmd *cobra.Command, args []string) error {
	root := "."
	if len(args) == 1 {
		root = args[0]
	}

	opts, err := buildWalkOptions()
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

func buildWalkOptions() (walker.Options, error) {
	after, err := parseTimeFlag(listAfter)
	if err != nil {
		return walker.Options{}, fmt.Errorf("解析 --after: %w", err)
	}
	before, err := parseTimeFlag(listBefore)
	if err != nil {
		return walker.Options{}, fmt.Errorf("解析 --before: %w", err)
	}

	return walker.Options{
		Exts:      listExts,
		NameGlob:  listNameGlob,
		NameRegex: listRegex,
		MinSize:   listMinSize,
		MaxSize:   listMaxSize,
		After:     after,
		Before:    before,
	}, nil
}

func parseTimeFlag(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("无法解析时间 %q，支持 RFC3339 或 2006-01-02", value)
}

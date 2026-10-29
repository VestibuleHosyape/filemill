package cmd

import (
	"fmt"
	"time"

	"filemill/internal/walker"

	"github.com/spf13/cobra"
)

// filterFlags holds shared walk/filter options used by multiple subcommands.
type filterFlags struct {
	exts     []string
	nameGlob string
	regex    string
	minSize  int64
	maxSize  int64
	after    string
	before   string
}

func (f *filterFlags) register(cmd *cobra.Command) {
	cmd.Flags().StringSliceVar(&f.exts, "ext", nil, "按扩展名过滤，可重复或逗号分隔，如 --ext .txt,.log")
	cmd.Flags().StringVar(&f.nameGlob, "name", "", "按文件名 glob 过滤，如 --name '*.log'")
	cmd.Flags().StringVar(&f.regex, "regex", "", "按文件名正则过滤")
	cmd.Flags().Int64Var(&f.minSize, "min-size", 0, "最小文件大小（字节）")
	cmd.Flags().Int64Var(&f.maxSize, "max-size", 0, "最大文件大小（字节）")
	cmd.Flags().StringVar(&f.after, "after", "", "仅包含此时间之后修改的文件，格式 2006-01-02 或 RFC3339")
	cmd.Flags().StringVar(&f.before, "before", "", "仅包含此时间之前修改的文件，格式 2006-01-02 或 RFC3339")
}

func (f *filterFlags) options() (walker.Options, error) {
	after, err := parseTimeFlag(f.after)
	if err != nil {
		return walker.Options{}, fmt.Errorf("解析 --after: %w", err)
	}
	before, err := parseTimeFlag(f.before)
	if err != nil {
		return walker.Options{}, fmt.Errorf("解析 --before: %w", err)
	}
	return walker.Options{
		Exts:      f.exts,
		NameGlob:  f.nameGlob,
		NameRegex: f.regex,
		MinSize:   f.minSize,
		MaxSize:   f.maxSize,
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

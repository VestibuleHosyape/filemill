package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	flagWorkers int
	flagDryRun  bool
	flagLogFile string

	logCloser io.Closer
)

// rootCmd is the base command for filemill.
var rootCmd = &cobra.Command{
	Use:   "filemill",
	Short: "单二进制、可组合的文件批量处理工具",
	Long: `filemill 是一个跨平台的文件批量处理 CLI。

支持递归遍历、过滤规则、并发处理，以及 hash / rename / dedupe 等子命令。
破坏性操作默认 dry-run，需显式 --apply 或 --yes 才会真正执行。`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return setupLogger(flagLogFile)
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if logCloser != nil {
			_ = logCloser.Close()
		}
	},
	SilenceUsage: true,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	defaultWorkers := runtime.NumCPU()
	if defaultWorkers > 8 {
		defaultWorkers = 8
	}
	if defaultWorkers < 1 {
		defaultWorkers = 1
	}

	rootCmd.PersistentFlags().IntVar(&flagWorkers, "workers", defaultWorkers, "并发 worker 数量，默认 min(NumCPU, 8)")
	rootCmd.PersistentFlags().BoolVar(&flagDryRun, "dry-run", false, "试运行：只打印将要执行的操作，不实际改动")
	rootCmd.PersistentFlags().StringVar(&flagLogFile, "log-file", "", "将结构化日志写入指定文件")
}

func setupLogger(path string) error {
	var w io.Writer = os.Stderr
	if path != "" {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return fmt.Errorf("打开日志文件: %w", err)
		}
		logCloser = f
		w = io.MultiWriter(os.Stderr, f)
	}

	handler := slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(handler))
	return nil
}

// Workers returns the configured worker count.
func Workers() int { return flagWorkers }

// DryRun returns whether dry-run mode is enabled.
func DryRun() bool { return flagDryRun }

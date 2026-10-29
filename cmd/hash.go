package cmd

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"filemill/internal/fsutil"
	"filemill/internal/hashutil"
	"filemill/internal/report"
	"filemill/internal/walker"
	"filemill/internal/worker"

	"github.com/spf13/cobra"
)

var (
	hashFilters filterFlags
	hashAlgo    string
	hashOutput  string
	hashOutFile string
)

type hashRecord struct {
	Path   string `json:"path"`
	Algo   string `json:"algo"`
	Hash   string `json:"hash"`
	Size   int64  `json:"size"`
	Error  string `json:"error,omitempty"`
	Failed bool   `json:"-"`
}

var hashCmd = &cobra.Command{
	Use:   "hash [path]",
	Short: "批量计算文件哈希",
	Long: `递归扫描目录（默认当前目录），并发计算匹配文件的哈希。

支持算法：md5 / sha1 / sha256。
输出格式：table（默认）/ csv / json。
单文件失败会记录错误并继续处理其余文件。`,
	Args: cobra.MaximumNArgs(1),
	RunE: runHash,
}

func init() {
	rootCmd.AddCommand(hashCmd)
	hashFilters.register(hashCmd)

	hashCmd.Flags().StringVar(&hashAlgo, "algo", hashutil.SHA256, "哈希算法：md5 / sha1 / sha256")
	hashCmd.Flags().StringVar(&hashOutput, "output", "table", "输出格式：table / csv / json")
	hashCmd.Flags().StringVar(&hashOutFile, "out", "", "将结果原子写入指定文件（默认打印到标准输出）")
}

func runHash(cmd *cobra.Command, args []string) error {
	root := "."
	if len(args) == 1 {
		root = args[0]
	}

	algo, err := hashutil.Normalize(hashAlgo)
	if err != nil {
		return err
	}
	format := strings.ToLower(strings.TrimSpace(hashOutput))
	switch format {
	case "table", "csv", "json":
	default:
		return fmt.Errorf("不支持的输出格式 %q（支持 table / csv / json）", hashOutput)
	}

	opts, err := hashFilters.options()
	if err != nil {
		return err
	}

	if DryRun() {
		return runHashDryRun(cmd, root, opts, algo)
	}

	stats := &report.Stats{}
	stats.Start()

	var (
		mu      sync.Mutex
		records []hashRecord
	)

	pool := worker.New(Workers())
	pool.Start()

	err = walker.Walk(root, opts, func(fi walker.FileInfo) error {
		stats.AddTotal(1)
		file := fi
		pool.SubmitErr(func() error {
			sum, hashErr := hashutil.File(file.Path, algo)
			rec := hashRecord{
				Path: file.Path,
				Algo: algo,
				Size: file.Size,
			}
			if hashErr != nil {
				rec.Failed = true
				rec.Error = hashErr.Error()
				stats.IncFailed()
				mu.Lock()
				records = append(records, rec)
				mu.Unlock()
				return fmt.Errorf("%s: %w", file.Path, hashErr)
			}
			rec.Hash = sum
			stats.IncSuccess()
			mu.Lock()
			records = append(records, rec)
			mu.Unlock()
			return nil
		}, nil)
		return nil
	})
	pool.Close()
	if err != nil {
		return err
	}

	sort.Slice(records, func(i, j int) bool { return records[i].Path < records[j].Path })

	var buf bytes.Buffer
	if err := writeHashRecords(&buf, records, format); err != nil {
		return err
	}

	if hashOutFile != "" {
		if err := fsutil.WriteFileAtomic(hashOutFile, buf.Bytes(), 0o644); err != nil {
			return err
		}
	} else if _, err := io.Copy(cmd.OutOrStdout(), &buf); err != nil {
		return err
	}

	return stats.WriteSummary(cmd.ErrOrStderr())
}

func runHashDryRun(cmd *cobra.Command, root string, opts walker.Options, algo string) error {
	stats := &report.Stats{}
	stats.Start()
	err := walker.Walk(root, opts, func(fi walker.FileInfo) error {
		stats.AddTotal(1)
		stats.IncSuccess()
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "dry-run\thash\t%s\t%s\n", algo, fi.Path)
		return err
	})
	if err != nil {
		return err
	}
	return stats.WriteSummary(cmd.ErrOrStderr())
}

func writeHashRecords(w io.Writer, records []hashRecord, format string) error {
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(records)
	case "csv":
		cw := csv.NewWriter(w)
		if err := cw.Write([]string{"path", "algo", "hash", "size", "error"}); err != nil {
			return err
		}
		for _, r := range records {
			if err := cw.Write([]string{r.Path, r.Algo, r.Hash, fmt.Sprintf("%d", r.Size), r.Error}); err != nil {
				return err
			}
		}
		cw.Flush()
		return cw.Error()
	default: // table
		for _, r := range records {
			if r.Failed {
				if _, err := fmt.Fprintf(w, "%s\t%s\tERROR\t%s\n", r.Path, r.Algo, r.Error); err != nil {
					return err
				}
				continue
			}
			if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", r.Path, r.Algo, r.Hash, r.Size); err != nil {
				return err
			}
		}
		return nil
	}
}
